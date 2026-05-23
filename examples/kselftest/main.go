package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/rlimit"
	"golang.org/x/sys/unix"
)

// insmod loads a kernel module from the given path using the finit_module(2)
// syscall. It returns nil if the module was loaded successfully or was already
// present in the kernel.
func insmod(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := unix.FinitModule(int(f.Fd()), "", 0); err != nil {
		if errors.Is(err, unix.EEXIST) {
			return nil // already loaded
		}
		return err
	}
	return nil
}

// testmodSkipReasons lists tests that are skipped only when bpf_testmod.ko has
// not been loaded. When the module is present, these tests are expected to pass.
var testmodSkipReasons = map[string]string{
	// These load successfully once bpf_testmod.ko is present.
	"fentry_recursive_target.bpf.o": "requires bpf_testmod.ko (bpf_testmod_fentry_testd)",
	"get_branch_snapshot.bpf.o":     "requires bpf_testmod.ko",
	"kfunc_call_destructive.bpf.o":  "requires bpf_testmod.ko",
	"kfunc_call_race.bpf.o":         "requires bpf_testmod.ko",
	"missed_kprobe.bpf.o":           "requires bpf_testmod.ko",
	"missed_kprobe_recursion.bpf.o": "requires bpf_testmod.ko",
	"sock_addr_kern.bpf.o":          "requires bpf_testmod.ko",
}

// skipReasons lists BPF object files whose failures are always expected,
// regardless of the kernel module state.
// Inspired by https://github.com/cilium/ebpf/blob/main/elf_reader_test.go
var skipReasons = map[string]string{
	// BTF map definition uses a nested map type not yet fully supported by
	// cilium/ebpf's BTF map parser.
	"bloom_filter_map.bpf.o": "BTF map definition not fully supported by cilium/ebpf (inner map type)",

	// MaxEntries is computed at runtime by test_progs; zero is invalid.
	"bpf_hashmap_lookup.bpf.o":      "MaxEntries is set to zero, requires test_progs runtime setup",
	"test_map_in_map_invalid.bpf.o": "MaxEntries is set to zero, requires test_progs runtime setup",
	"test_mmap.bpf.o":               "MaxEntries is set to zero, requires test_progs runtime setup",
	"test_ringbuf_map_key.bpf.o":    "MaxEntries is set to zero, requires test_progs runtime setup",
	"test_ringbuf_n.bpf.o":          "MaxEntries is set to zero, requires test_progs runtime setup",
	"test_ringbuf_write.bpf.o":      "MaxEntries is set to zero, requires test_progs runtime setup",

	// Ring buffer size must be a multiple of the page size; value is set by test_progs.
	"test_ringbuf_multi.bpf.o": "ring buffer size must be a page-size multiple, set by test_progs",

	// Requires the kernel to be built with CONFIG_BPF_KPROBE_OVERRIDE.
	"kprobe_multi_override.bpf.o": "requires CONFIG_BPF_KPROBE_OVERRIDE",

	// Reference to a symbol defined in a sibling object file (cross-ELF).
	"kptr_xchg_inline.bpf.o":     "references symbol defined in a sibling object file",
	"stacktrace_ips.bpf.o":       "references symbol defined in a sibling object file",
	"test_fill_link_info.bpf.o":  "references symbol defined in a sibling object file",
	"test_subskeleton.bpf.o":     "cross-ELF linked object, requires test_subskeleton_lib",
	"test_subskeleton_lib.bpf.o": "cross-ELF linked object, referenced by test_subskeleton",

	// kfunc_module_order needs bpf_test_modorder_x.ko and bpf_test_modorder_y.ko,
	// not covered by loading bpf_testmod.ko alone.
	"kfunc_module_order.bpf.o": "requires bpf_test_modorder_x.ko and bpf_test_modorder_y.ko",

	// Kernel ksym not present or not exported in BTF in this build.
	"test_ksyms.bpf.o":                 "references kernel symbol (bpf_link_fops) not in BTF",
	"test_ksyms_btf_write_check.bpf.o": "references kernel symbol not in BTF",
	"normal_map_btf.bpf.o":             "references BTF struct type absent from this kernel build",

	// Intentionally invalid BPF programs — negative tests.
	"local_kptr_stash_fail.bpf.o":            "intentionally invalid kptr access",
	"loop3.bpf.o":                            "program intentionally exceeds BPF instruction limit",
	"priv_freplace_prog.bpf.o":               "freplace program requires a target program fd",
	"rbtree_btf_fail__add_wrong_type.bpf.o":  "intentionally invalid rbtree type usage",
	"rbtree_btf_fail__wrong_node_type.bpf.o": "intentionally invalid rbtree node type",
	"stream_fail.bpf.o":                      "intentionally invalid BPF program (vprintk string arg)",
	"strncmp_test.bpf.o":                     "intentionally invalid BPF program (non-const string size)",
	"tailcall_freplace.bpf.o":                "freplace program requires a target program fd",
	"test_autoload.bpf.o":                    "intentionally exercises failed program attachment",
	"test_get_stack_rawtp_err.bpf.o":         "intentionally invalid BPF program",
	"test_global_func7.bpf.o":                "intentionally invalid global function call",
	"test_global_func10.bpf.o":               "intentionally invalid global function call",
	"test_global_func11.bpf.o":               "intentionally invalid global function call",
	"test_global_func12.bpf.o":               "intentionally invalid global function call",
	"test_global_func13.bpf.o":               "intentionally invalid global function call",
	"test_global_func14.bpf.o":               "intentionally invalid global function call",
	"test_global_func17.bpf.o":               "intentionally invalid global function call",
	"test_log_buf.bpf.o":                     "intentionally invalid BPF program (tests log buffer)",
	"test_log_fixup.bpf.o":                   "uses missing kfunc to test error message fixup",
	"test_map_in_map.bpf.o":                  "HashOfMaps requires InnerMap definition",
	"test_pinning.bpf.o":                     "requires MapOptions.PinPath, set by test_progs",
	"test_pinning_devmap.bpf.o":              "requires MapOptions.PinPath, set by test_progs",
	"test_pinning_invalid.bpf.o":             "intentionally invalid map pinning path",
	"test_sockmap_invalid_update.bpf.o":      "intentionally invalid sockmap program",
	"test_tp_btf_nullable.bpf.o":             "requires specific raw tracepoint BTF annotation",
	"test_trace_ext.bpf.o":                   "freplace program requires a target program fd",
	"test_xdp_bpf2bpf.bpf.o":                 "fexit requires a target program fd",
	"test_xdp_devmap_helpers.bpf.o":          "intentionally invalid devmap helper access",
	"test_xdp_devmap_tailcall.bpf.o":         "intentionally invalid devmap tailcall",
	"timer_failure.bpf.o":                    "intentionally invalid async callback return type",
	"timer_mim_reject.bpf.o":                 "intentionally invalid timer/map-in-map combination",
	"tracing_failure.bpf.o":                  "intentionally invalid tracing program attachment",
	"uninit_stack.bpf.o":                     "intentionally invalid BTF (uninitialized stack)",
	"xdp_metadata2.bpf.o":                    "metadata kfuncs require a device, not available in test VM",

	// cilium/ebpf requires variable ksyms to be represented as *Void in BTF,
	// but these module-exported ksyms are typed as *btf.Var. The library
	// returns "not *btf.Var: not supported" at the LoadCollectionSpec stage.
	"kprobe_multi_session.bpf.o":       "cilium/ebpf: bpf_fentry_test1 is *btf.Var, not *Void (not supported)",
	"ksym_race.bpf.o":                  "cilium/ebpf: bpf_testmod_ksym_percpu is *btf.Var, not *Void (not supported)",
	"read_bpf_task_storage_busy.bpf.o": "cilium/ebpf: bpf_task_storage_busy is *btf.Var, not *Void (not supported)",

	// Still fail with bpf_testmod.ko loaded: kernel-level feature mismatch.
	"btf_type_tag_user.bpf.o":    "Cannot access kernel struct bpf_testmod_btf_type_tag (BTF tag layout mismatch)",
	"fexit_bpf2bpf_simple.bpf.o": "fexit not supported on test_pkt_md_access",
	"fmod_ret_freplace.bpf.o":    "fmod_ret not supported on security_new_get_constant",
	"raw_tp_null_fail.bpf.o":     "intentionally invalid raw_tp program",

	// struct_ops programs that need the ops type registered via test infrastructure,
	// not just the module loaded. Fail with "operation not supported" regardless.
	"struct_ops_autocreate.bpf.o":          "struct_ops type requires test_run registration",
	"struct_ops_autocreate2.bpf.o":         "struct_ops type requires test_run registration",
	"struct_ops_private_stack.bpf.o":       "struct_ops type requires test_run registration",
	"struct_ops_private_stack_fail.bpf.o":  "struct_ops type requires test_run registration",
	"struct_ops_private_stack_recur.bpf.o": "struct_ops type requires test_run registration",
	"trace_dummy_st_ops.bpf.o":             "struct_ops type requires test_run registration",
}

// skipReason returns a non-empty string when the given filename is expected to
// not load successfully. testmodLoaded indicates whether bpf_testmod.ko has
// been loaded into the kernel.
func skipReason(name string, testmodLoaded bool) string {
	if r, ok := skipReasons[name]; ok {
		return r
	}
	if !testmodLoaded {
		if r, ok := testmodSkipReasons[name]; ok {
			return r
		}
	}
	switch {
	// verifier_*.bpf.o files are BPF verifier self-tests. They contain
	// intentionally invalid programs designed to exercise the verifier's
	// rejection logic and are therefore not expected to load.
	case strings.HasPrefix(name, "verifier_"):
		return "verifier test file — contains intentionally invalid programs"

	// bad_* files are explicitly crafted to be rejected.
	case strings.HasPrefix(name, "bad_"):
		return "intentionally malformed BPF object"

	// freplace_* programs replace an already-loaded program; they cannot be
	// loaded standalone.
	case strings.HasPrefix(name, "freplace_"):
		return "freplace program requires a target program fd"

	// linked_* objects are designed to be linked together; neither loads alone.
	case strings.HasPrefix(name, "linked_"):
		return "cross-ELF linked object, requires sibling file"

	// dummy_st_ops_* test struct_ops programs. The ops type registration
	// requires more than just loading the module so these always fail standalone.
	case strings.HasPrefix(name, "dummy_st_ops"):
		return "struct_ops type requires test_run registration"

	// fentry_recursive.bpf.o tests recursive fentry, which is intentionally
	// unsupported. Use exact match so fentry_recursive_target.bpf.o (handled
	// via testmodSkipReasons) can run when bpf_testmod.ko is loaded.
	case name == "fentry_recursive.bpf.o":
		return "recursive fentry attachment is not supported"

	// rbtree_btf_fail__* are intentionally invalid rbtree usage tests.
	case strings.HasPrefix(name, "rbtree_btf_fail__"):
		return "intentionally invalid rbtree usage"

	// test_core_reloc_* use companion btf__core_reloc_*.bpf.o data files to
	// supply target BTF types; they cannot load standalone.
	case strings.HasPrefix(name, "test_core_reloc_"):
		return "CO-RE relocation test requires companion btf__core_reloc_* data file"
	}
	return ""
}

func collectFiles(args []string) ([]string, error) {
	if len(args) == 0 {
		return filepath.Glob("*.bpf.o")
	}
	var out []string
	for _, arg := range args {
		matches, err := filepath.Glob(arg)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", arg, err)
		}
		if len(matches) == 0 {
			out = append(out, arg)
		} else {
			out = append(out, matches...)
		}
	}
	return out, nil
}

func main() {
	// Remove the RLIMIT_MEMLOCK restriction so BPF map and program creation
	// works even when the kernel enforces a low memlock limit.
	if err := rlimit.RemoveMemlock(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to remove memlock limit: %v\n", err)
	}

	// Load bpf_testmod.ko if present so that tests depending on its kfuncs,
	// ksyms, and struct_ops types can run rather than being skipped.
	testmodLoaded := false
	if err := insmod("bpf_testmod.ko"); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "# warning: loading bpf_testmod.ko: %v\n", err)
		}
	} else {
		testmodLoaded = true
		fmt.Println("# bpf_testmod.ko loaded")
	}

	files, err := collectFiles(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "no BPF object files found\n")
		os.Exit(1)
	}

	fmt.Printf("1..%d\n", len(files))

	failures := 0
	for i, f := range files {
		name := filepath.Base(f)

		// Check the skip list before any loading attempt so that files which
		// fail at the ELF/BTF parse stage are also handled correctly.
		if reason := skipReason(name, testmodLoaded); reason != "" {
			fmt.Printf("ok %3d - %s # SKIP %s\n", i+1, name, reason)
			continue
		}

		spec, err := ebpf.LoadCollectionSpec(f)
		if err != nil {
			fmt.Printf("not ok %3d - %s: %v\n", i+1, name, err)
			failures++
			continue
		}

		coll, err := ebpf.NewCollection(spec)
		if err != nil {
			fmt.Printf("not ok %3d - %s: %v\n", i+1, name, err)
			failures++
			continue
		}
		coll.Close()

		fmt.Printf("ok %3d - %s (%d programs, %d maps)\n", i+1, name, len(spec.Programs), len(spec.Maps))
	}

	if failures > 0 {
		os.Exit(1)
	}
}
