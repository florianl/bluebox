package initramfs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBluebox(t *testing.T) {
	b := New()
	if err := b.Generate(io.Discard); err != nil {
		t.Fatal(err)
	}
}

func TestExecuteOrder(t *testing.T) {
	dir := t.TempDir()
	b := New()

	var want []execEntry

	for i := range 10 {
		path := filepath.Join(dir, fmt.Sprintf("exec%d", i))
		if err := os.WriteFile(path, nil, 0o755); err != nil {
			t.Fatal(err)
		}

		args := []string{fmt.Sprintf("-n=%d", i)}
		if err := b.Execute(path, args...); err != nil {
			t.Fatal(err)
		}

		want = append(want, execEntry{path: path, args: args})
	}

	if !reflect.DeepEqual(b.execs, want) {
		t.Errorf("execs = %v, want %v", b.execs, want)
	}
}

func TestExecuteDuplicate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "exec")
	if err := os.WriteFile(path, nil, 0o755); err != nil {
		t.Fatal(err)
	}

	b := New()
	if err := b.Execute(path); err != nil {
		t.Fatal(err)
	}

	if err := b.Execute(path); err == nil {
		t.Error("Execute accepted the same executable twice")
	}

	if err := b.Embed(path); err == nil {
		t.Error("Embed accepted a file that is already executed")
	}
}

func TestBaseNameCollision(t *testing.T) {
	dir := t.TempDir()

	var paths []string
	for _, sub := range []string{"a", "b"} {
		path := filepath.Join(dir, sub, "x.test")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0o755); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}

	b := New()
	if err := b.Execute(paths[0]); err != nil {
		t.Fatal(err)
	}
	if err := b.Execute(paths[1]); err == nil {
		t.Error("Execute accepted two executables with the same base name")
	}
	if err := b.Embed(paths[1]); err == nil {
		t.Error("Embed accepted a file with the base name of an executable")
	}

	b = New()
	if err := b.Embed(paths[0]); err != nil {
		t.Fatal(err)
	}
	if err := b.Embed(paths[1]); err == nil {
		t.Error("Embed accepted two files with the same base name")
	}
	if err := b.Execute(paths[1]); err == nil {
		t.Error("Execute accepted an executable with the base name of an embedded file")
	}
}

func TestReservedNames(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{"init", "bluebox", "bluebox-init"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, nil, 0o755); err != nil {
			t.Fatal(err)
		}

		b := New()
		if err := b.Execute(path); err == nil {
			t.Errorf("Execute accepted reserved name %q", path)
		}
		if err := b.Embed(path); err == nil {
			t.Errorf("Embed accepted reserved name %q", path)
		}
	}
}

// trailerFailWriter fails the write that carries the trailer of a cpio archive.
type trailerFailWriter struct{}

var errTrailer = errors.New("no space left for trailer")

func (trailerFailWriter) Write(p []byte) (int, error) {
	if bytes.Contains(p, []byte("TRAILER!!!")) {
		return 0, errTrailer
	}
	return len(p), nil
}

func TestGenerateTrailerError(t *testing.T) {
	b := New()
	if err := b.Generate(trailerFailWriter{}); err == nil {
		t.Fatal("Generate dropped the error from writing the trailer")
	}
}

func TestGenerateReproducible(t *testing.T) {
	dir := t.TempDir()

	var archives [2]bytes.Buffer
	for i := range archives {
		b := New()
		for j := range 10 {
			path := filepath.Join(dir, fmt.Sprintf("file%d", j))
			if err := os.WriteFile(path, []byte(path), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := b.Embed(path); err != nil {
				t.Fatal(err)
			}
		}
		if err := b.Generate(&archives[i]); err != nil {
			t.Fatal(err)
		}
	}

	if !bytes.Equal(archives[0].Bytes(), archives[1].Bytes()) {
		t.Error("the same input resulted in different archives")
	}
}
