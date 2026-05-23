# kselftest

`kselftest` is a selftest runner that demonstrates how bluebox can execute
Linux BPF kernel selftests inside a minimal QEMU virtual machine.

It loads pre-compiled BPF object files (`.bpf.o`) from
[`ghcr.io/cilium/ci-kernels:stable-selftests`](https://github.com/cilium/ci-kernels/pkgs/container/ci-kernels?tag=stable-selftests)
into the matching kernel and reports the result. The approach is inspired by
[`elf_reader_test.go`](https://github.com/cilium/ebpf/blob/main/elf_reader_test.go)
in cilium/ebpf.

## How it works

The CI job (`.github/workflows/example-selftests.yml`) drives the full pipeline:

1. Extract the `stable-selftests` OCI image — it bundles a pre-compiled kernel
   and the matching BPF object files (`tools/testing/selftests/bpf/*.bpf.o`).
2. Cross-compile this binary for `linux/amd64` with `CGO_ENABLED=0`.
3. Use bluebox to pack the binary, `bpf_testmod.ko`, and the `.bpf.o` files
   into an `initramfs.cpio` archive.
4. Boot the kernel in QEMU with that archive as the initial ramdisk.
5. At boot the runner loads `bpf_testmod.ko` via `finit_module(2)`, then
   iterates over every embedded `.bpf.o` file and attempts to load it into the
   kernel using [`github.com/cilium/ebpf`](https://github.com/cilium/ebpf).

## CI job

The workflow runs on a weekly schedule (Saturday 08:15 UTC).  Find it under **Actions → bluebox CI/CD
kselftest example** in the GitHub repository.

## Usage

```
kselftest [pattern-or-path ...]
```

Arguments are treated as glob patterns or literal file paths.  With no
arguments the runner globs `*.bpf.o` in the current directory.

```
# Run all BPF object files in the current directory
kselftest

# Run a specific subset
kselftest verifier_*.bpf.o atomics.bpf.o
```

## Skipped tests

Not every BPF object file can be loaded standalone.  The runner skips known
cases with a `# SKIP <reason>` annotation rather than failing.  There are
four categories:

### 1. Verifier negative tests (`verifier_*.bpf.o`)

These files are part of the kernel's BPF verifier test suite.  They contain
programs that are deliberately invalid — the point is to verify that the
verifier *rejects* them.  Loading them with `cilium/ebpf` without the
`test_progs` orchestration layer would always fail, so they are skipped as a
group via prefix rule.

### 2. Requires `bpf_testmod.ko`

Some tests reference kfuncs, ksyms, or tracing targets exported by the
`bpf_testmod.ko` out-of-tree test module that ships inside the
`stable-selftests` image.  The runner loads that module at startup via
`insmod("bpf_testmod.ko")`.  Tests in `testmodSkipReasons` are skipped only
when the module could not be loaded; they are expected to pass once it is
present.

Tests that still fail after loading the module are listed in the static
`skipReasons` map with an explanation.  The current known cases are:

- **`*btf.Var` ksyms** — `cilium/ebpf` requires variable ksyms to be
  represented as `*Void` in BTF, but the module exports them as `*btf.Var`.
  The library returns `not *btf.Var: not supported` at the
  `LoadCollectionSpec` stage regardless of library version (present as of
  `cilium/ebpf v0.21.0`).
- **struct_ops programs** — loading a `struct_ops` BPF program requires the
  ops type to be registered via test-framework infrastructure beyond a bare
  `insmod`.

### 3. Intentionally invalid programs

Several files outside the `verifier_*` group are also designed to fail:
`freplace_*` programs that require a target fd, `linked_*` objects that
require a sibling file, programs with zero `MaxEntries` that test_progs sets
at runtime, programs exercising invalid kptr/rbtree/sockmap usage, and so on.
Each entry in `skipReasons` documents the exact reason.

### 4. Missing kernel or library support

A small number of tests require features absent from this environment:
`CONFIG_BPF_KPROBE_OVERRIDE`, device-attached metadata kfuncs, pinned maps
(no `PinPath` set), or kernel BTF symbols that are not exported in this
build.
