package initramfs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"text/template"

	exec "golang.org/x/sys/execabs"
)

// defaultEnvironment is the minimal environment init sets up for the executables.
var defaultEnvironment = []environment{
	mount{source: "devtmpfs", target: "/dev", fstype: "devtmpfs", flags: 0, data: "", targetPerm: 0o755, targetCreate: true},
	mount{source: "tmpfs", target: "/tmp", fstype: "tmpfs", flags: 0, data: "", targetPerm: 0o755, targetCreate: true},
	mount{source: "proc", target: "/proc", fstype: "proc", flags: 0, data: "", targetPerm: 0o555, targetCreate: true},
	nod{path: "/dev/tty", mode: syscall.S_IFCHR | 0o666, dev: 0x0500},
	nod{path: "/dev/urandom", mode: syscall.S_IFCHR | 0o444, dev: 0x0109},
	mount{source: "sysfs", target: "/sys", fstype: "sysfs", flags: 0, data: "", targetPerm: 0o555, targetCreate: true},
	mount{source: "securityfs", target: "/sys/kernel/security", fstype: "securityfs", flags: 0, data: ""},
	mount{source: "debugfs", target: "/sys/kernel/debug", fstype: "debugfs", flags: 0, data: ""},
	mount{source: "bpffs", target: "/sys/fs/bpf", fstype: "bpf", flags: 0, data: ""},
}

// renderInit writes the Go source of init, which sets up env, to w.
func renderInit(w io.Writer, env []environment) error {
	tmpl, err := template.New("").Parse(initTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}
	if err := tmpl.Execute(w, initTemplateConfig{Environment: env}); err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	return nil
}

// renderBluebox writes the Go source of bluebox-init to w.
func (b *Bluebox) renderBluebox(w io.Writer) error {
	config := blueboxTemplateConfig{}

	for _, e := range b.execs {
		config.Executables = append(config.Executables, filepath.Base(e.path))
		config.Arguments = append(config.Arguments, e.args)
	}

	for _, env := range b.envVars {
		config.EnvVars = append(config.EnvVars, envVar{
			Key:   env.Key,
			Value: env.Value,
		})
	}

	tmpl, err := template.New("").Parse(blueboxTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}
	if err := tmpl.Execute(w, config); err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	return nil
}

// createInit writes a Go program and compiles it so it can be used as init.
func (b *Bluebox) createInit(dir string) error {
	f, err := os.OpenFile(filepath.Join(dir, "init.go"), os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}

	if err := renderInit(f, defaultEnvironment); err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	return b.goBuild(filepath.Join(dir, "init"), f.Name())
}

// goBuild compiles the Go file src for the target architecture and writes the executable to out.
func (b *Bluebox) goBuild(out, src string) error {
	path, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("failed to look up 'go' executable: %v", err)
	}

	cmd := exec.CommandContext(context.Background(), path, "build", "-trimpath", "-o", out, src)

	// Settings of the host must not leak into the build. The executables have to be statically
	// linked and flags from GOFLAGS could change or break the build.
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GOARCH=%s", b.arch),
		"GOOS=linux",
		"CGO_ENABLED=0",
		"GOFLAGS=",
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go build failed: %v\n%s", err, bytes.TrimSpace(output))
	}

	return nil
}

// createBluebox writes a Go program and compiles it. In a sequential order it will execute
// the given execs with their respective args.
func (b *Bluebox) createBluebox(tmpDir string) error {
	f, err := os.OpenFile(filepath.Join(tmpDir, "bluebox.go"), os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %v", err)
	}

	if err := b.renderBluebox(f); err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %v", err)
	}

	return b.goBuild(filepath.Join(tmpDir, "bluebox-init"), f.Name())
}
