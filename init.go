package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

var initFile string = `package main

import (
	"fmt"
	"os"
	"io"
	"os/exec"
	"strings"
	"syscall"
)

func printClose(r io.ReadCloser, prefix string){
	defer r.Close()
	slurp, _ := io.ReadAll(r)
	if len(slurp) == 0 {
		return
	}
	fmt.Printf("[            ] %s\n%s", prefix, slurp)
}


func main() {
	// Create a minimal environment for the Linux kernel
	os.MkdirAll("/dev", 0o777)
	fmt.Println("[            ]\tos.MkdirAll(\"/dev\", 0o777)")

	syscall.Mount("devtmpfs", "/dev", "devtmpfs", uintptr(0), "")
	fmt.Println("[            ]\tsyscall.Mount(\"devtmpfs\", \"/dev\", \"devtmpfs\", uintptr(0), \"\")")

	os.MkdirAll("/tmp", 0o777)
	fmt.Println("[            ]\tos.MkdirAll(\"/tmp\", 0o777)")


	os.MkdirAll("/proc", 0o555)
	fmt.Println("[            ]\tos.MkdirAll(\"/proc\", 0o555)")

	syscall.Mount("proc", "/proc", "proc", uintptr(0), "")
	fmt.Println("[            ]\tsyscall.Mount(\"proc\", \"/proc\", \"proc\", uintptr(0), \"\")")

	syscall.Mount("tmpfs", "/tmp", "tmpfs", uintptr(0), "")
	fmt.Println("[            ]\tsyscall.Mount(\"tmpfs\", \"/tmp\", \"tmpfs\", uintptr(0), \"\")")

	os.Remove("/dev/tty")
	syscall.Mknod("/dev/tty", syscall.S_IFCHR | 0o666, 0x0500)
	fmt.Println("[            ]\tsyscall.Mknod(\"/dev/tty\", syscall.S_IFCHR | 0o666, 0x0500)")


	os.Remove("/dev/urandom")
	syscall.Mknod("/dev/urandom", syscall.S_IFCHR | 0o444, 0x0109)
	fmt.Println("[            ]\tsyscall.Mknod(\"/dev/urandom\", syscall.S_IFCHR | 0o444, 0x0109)")

	os.MkdirAll("/sys", 0o555)
	fmt.Println("[            ]\tos.MkdirAll(\"/sys\", 0o555)")

	syscall.Mount("sysfs", "/sys", "sysfs", uintptr(0), "")
	fmt.Println("[            ]\tsyscall.Mount(\"sysfs\", \"/sys\", \"sysfs\", uintptr(0), \"\")")

	syscall.Mount("securityfs", "/sys/kernel/security", "securityfs", uintptr(0), "")
	fmt.Println("[            ]\tsyscall.Mount(\"securityfs\", \"/sys/kernel/security\", \"securityfs\", uintptr(0), \"\")")

	syscall.Mount("debugfs", "/sys/kernel/debug", "debugfs", uintptr(0), "")
	fmt.Println("[            ]\tsyscall.Mount(\"debugfs\", \"/sys/kernel/debug\", \"debugfs\", uintptr(0), \"\")")

	syscall.Mount("bpffs", "/sys/fs/bpf", "bpf", uintptr(0), "")
	fmt.Println("[            ]\tsyscall.Mount(\"bpffs\", \"/sys/fs/bpf\", \"bpf\", uintptr(0), \"\")")

	// Execute the testing executables
	for i, exe := range execs {
		cmd := exec.Command(fmt.Sprintf("./%s", exe), strings.Join(exeArg[i], ", "))
		fmt.Printf("[            ]\t%s\n", cmd.Path)
		stderr, err := cmd.StderrPipe()
		if err != nil {
			fmt.Printf("[            ]\tFailed to redirect stderr for '%s': %v\n", err, exe)
			continue
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Printf("[            ]\tFailed to redirect stdout for '%s': %v\n", err, exe)
			continue
		}
		if err := cmd.Start(); err != nil {
			fmt.Printf("[            ]\tFailure starting %s: %v\n", exe, err)
			printClose(stderr, "stderr")
			stdout.Close()
			continue
		}
		for {
			var s syscall.WaitStatus
			var r syscall.Rusage
			if p, err := syscall.Wait4(-1, &s, 0, &r); p == cmd.Process.Pid {
				fmt.Printf("[            ]\t%s exited, exit status %d\n", cmd.Path, s.ExitStatus())
				printClose(stderr, "stderr")
				break
			} else if p != -1 {
				fmt.Printf("[            ]\tReaped PID %d, exit status %d\n", p, s.ExitStatus())
				printClose(stderr, "stderr")
				break
			} else {
				fmt.Printf("[            ]\tError from Wait4 for orphaned child: %v\n", err)
				printClose(stderr, "stderr")
				break
			}
		}

		if err := cmd.Process.Release(); err != nil {
			fmt.Printf("[            ]\tError releasing process %v: %v\n", cmd, err)
		}
		printClose(stdout, "stdout")
	}

	// Shut VM down
	if err := syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF); err != nil {
		fmt.Printf( "[            ]\tPower off failed: %v\n", err)
	}
}

`

// createInit writes a Go program and compiles it so it can be used as init.
func createInit(execs []string, args [][]string) (string, error) {
	dir, err := os.MkdirTemp("", "go-init")
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.OpenFile(filepath.Join(dir, "main.go"), os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		os.RemoveAll(dir)
		log.Fatal(err)
	}
	if _, err := f.Write([]byte(initFile)); err != nil {
		f.Close()
		os.RemoveAll(dir)
		log.Fatal(err)
	}

	if _, err := f.Write([]byte("var execs []string= []string{\n")); err != nil {
		f.Close()
		os.RemoveAll(dir)
		log.Fatal(err)
	}

	for _, exec := range execs {
		if _, err := f.Write([]byte(fmt.Sprintf("\t\"%s\",\n", filepath.Base(exec)))); err != nil {
			f.Close()
			os.RemoveAll(dir)
			log.Fatal(err)
		}
	}

	if _, err := f.Write([]byte("}\n")); err != nil {
		f.Close()
		os.RemoveAll(dir)
		log.Fatal(err)
	}

	if _, err := f.Write([]byte("var exeArg [][]string= [][]string{\n")); err != nil {
		f.Close()
		os.RemoveAll(dir)
		log.Fatal(err)
	}

	for _, arg := range args {
		var exeArg string
		for _, a := range arg {
			if len(exeArg) == 0 {
				exeArg += fmt.Sprintf("\"%s\"", a)
			} else {
				exeArg += fmt.Sprintf(", \"%s\"", a)
			}
		}
		if _, err := f.Write([]byte(fmt.Sprintf("\t{%s},\n", exeArg))); err != nil {
			f.Close()
			os.RemoveAll(dir)
			log.Fatal(err)
		}
	}

	if _, err := f.Write([]byte("}\n")); err != nil {
		f.Close()
		os.RemoveAll(dir)
		log.Fatal(err)
	}

	if err := f.Close(); err != nil {
		os.RemoveAll(dir)
		log.Fatal(err)
	}

	cmd := exec.CommandContext(context.TODO(), "go", "build", "-o", filepath.Join(dir, "init"),
		f.Name())

	if arch != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("GOARCH=%s", arch))
	}

	if err := cmd.Run(); err != nil {
		os.RemoveAll(dir)
		log.Fatal(err)
	}

	return dir, nil
}
