package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"text/template"

	exec "golang.org/x/sys/execabs"
)

// createInit writes a Go program and compiles it so it can be used as init.
func createInit(dir string) error {
	f, err := os.OpenFile(filepath.Join(dir, "init.go"), os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	config := Bluebox{
		Arguments: args,
		Environment: []Environment{
			Mount{source: "devtmpfs", target: "/dev", fstype: "devtmpfs", flags: 0, data: "", targetPerm: 0o755, targetCreate: true},
			Mount{source: "tmpfs", target: "/tmp", fstype: "tmpfs", flags: 0, data: "", targetPerm: 0o755, targetCreate: true},
			Mount{source: "proc", target: "/proc", fstype: "proc", flags: 0, data: "", targetPerm: 0o555, targetCreate: true},
			Nod{path: "/dev/tty", mode: syscall.S_IFCHR | 0o666, dev: 0x0500},
			Nod{path: "/dev/urandom", mode: syscall.S_IFCHR | 0o444, dev: 0x0109},
			Mount{source: "sysfs", target: "/sys", fstype: "sysfs", flags: 0, data: "", targetPerm: 0o555, targetCreate: true},
			Mount{source: "securityfs", target: "/sys/kernel/security", fstype: "securityfs", flags: 0, data: ""},
			Mount{source: "debugfs", target: "/sys/kernel/debug", fstype: "debugfs", flags: 0, data: ""},
			Mount{source: "bpffs", target: "/sys/fs/bpf", fstype: "bpf", flags: 0, data: ""},
		},
	}

	config.Executables = make([]string, len(execs))
	// Strip the path from the executable before adding it.
	for i, exe := range execs {
		config.Executables[i] = filepath.Base(exe)
	}

	tmpl, err := template.New("").Parse(initTemplate)
	if err != nil {
		return err
	}
	if err := tmpl.Execute(f, config); err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	cmd := exec.CommandContext(context.TODO(), "go", "build", "-o", filepath.Join(dir, "init"),
		f.Name())

	if arch != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("GOARCH=%s", arch))
	}

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

// createBluebox writes a Go program and compiles it. In a sequential order it will execute
// the given execs with their respective args.
func createBluebox(dir string, execs []string, args [][]string) error {
	f, err := os.OpenFile(filepath.Join(dir, "bluebox.go"), os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	config := Bluebox{
		Arguments: args,
	}

	config.Executables = make([]string, len(execs))
	// Strip the path from the executable before adding it.
	for i, exe := range execs {
		config.Executables[i] = filepath.Base(exe)
	}

	tmpl, err := template.New("").Parse(blueboxTemplate)
	if err != nil {
		return err
	}
	if err := tmpl.Execute(f, config); err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	cmd := exec.CommandContext(context.TODO(), "go", "build", "-o", filepath.Join(dir, "bluebox-init"),
		f.Name())

	if arch != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("GOARCH=%s", arch))
	}

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
