// Package initramfs provides functionality to generate an archive, which can be extracted by the
// Linux kernel when the kernel boots up.
package initramfs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/cavaliergopher/cpio"
)

type envVar struct {
	Key, Value string
}

type Bluebox struct {
	// arch holds the GOARCH value used when compiling the init.
	arch string

	// enVars holds a list of environment variables.
	envVars []envVar

	// execs maps executables to their arguments.
	execs map[string][]string

	// embeddings maps a list of files that will be added additionallity into the resulting arichve.
	embeddings map[string]bool
}

// New constructs Bluebox with default values.
func New() *Bluebox {
	return &Bluebox{
		arch: runtime.GOARCH,
	}
}

// Execute embeds executable into the resulting archive and passes arg as arguments to
// its execution instruction.
func (b *Bluebox) Execute(executable string, args ...string) error {
	if executable == "init" || executable == "bluebox" || executable == "bluebox-init" {
		return fmt.Errorf("embedded executable should not be named '%s'", executable)
	}

	// TODO: Validate if and how executing the same executable is possible.
	if _, ok := b.execs[executable]; ok {
		return fmt.Errorf("%s is already embeded. Can not add it multiple times", executable)
	}

	// Cross check with read-only files.
	if _, ok := b.embeddings[executable]; ok {
		return fmt.Errorf("%s is already embeded. Can not add it multiple times", executable)
	}

	// Verify name references a file.
	s, err := os.Stat(executable)
	if err != nil {
		return err
	}
	if s.IsDir() {
		return fmt.Errorf("%s should not be a directory", executable)
	}

	b.execs[executable] = args[:]

	return nil
}

// Embed adds file into the resulting archive but does not add it for execution by the init program.
func (b *Bluebox) Embed(file string) error {
	if _, ok := b.embeddings[file]; ok {
		return fmt.Errorf("%s is already embeded. Can not add it multiple times", file)
	}

	// Cross check with executable files.
	if _, ok := b.execs[file]; ok {
		return fmt.Errorf("%s is already embeded. Can not add it multiple times", file)
	}

	// Verify name references a file.
	s, err := os.Stat(file)
	if err != nil {
		return err
	}
	if s.IsDir() {
		return fmt.Errorf("%s should not be a directory", file)
	}

	b.embeddings[file] = true

	return nil
}

// Setarch sets the architecture for the generated initramfs archive. If the architecture is not
// part of GOARCH an error will be returned. By default the architecture of the host is used.
func (b *Bluebox) Setarch(arch string) error {
	// TODO: validate arch.
	b.arch = arch
	return nil
}

// Setenv sets the value of the environment variable named by the key.
func (b *Bluebox) Setenv(key, value string) {
	b.envVars = append(b.envVars,
		envVar{Key: key,
			Value: value})
}

// Generate writes the configured initramfs archive to a file. Otherwise an error is returned.
// To do so it first auto generates a init program from the given parameters and compiles it before
// placing it into archive.
func (b *Bluebox) Generate(archive io.Writer) error {
	tmpDir, err := os.MkdirTemp("", "bluebox-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// Generate the init executable that is called by the kernel and prepares the system for
	// further use.
	if err := b.createInit(tmpDir); err != nil {
		return err
	}

	// Generate bluebox-init which will call the given executables in a sequential order.
	if err := b.createBluebox(tmpDir); err != nil {
		return err
	}

	w := cpio.NewWriter(archive)
	defer w.Close()

	// Add init to archive.
	if err := addFile(w, filepath.Join(tmpDir, "init")); err != nil {
		return err
	}

	// Add bluebox-init to archive.
	if err := addFile(w, filepath.Join(tmpDir, "bluebox-init")); err != nil {
		return err
	}

	for file := range b.execs {
		if err := addFile(w, file); err != nil {
			return err
		}
	}

	for file := range b.embeddings {
		if err := addFile(w, file); err != nil {
			return err
		}
	}

	return nil
}

// addFile adds file to the cpio archive.
func addFile(w *cpio.Writer, file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	if err := w.WriteHeader(&cpio.Header{
		Name: filepath.Base(file),
		Mode: cpio.FileMode(fi.Mode().Perm()),
		Size: fi.Size(),
	}); err != nil {
		return err
	}
	if _, err := io.Copy(w, f); err != nil {
		return err
	}
	return w.Flush()
}
