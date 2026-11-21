package initramfs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/cavaliergopher/cpio"
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

func TestGenerateArchiveContent(t *testing.T) {
	dir := t.TempDir()

	files := []struct {
		name    string
		content string
		perm    os.FileMode
		exec    bool
	}{
		{name: "second.test", content: "second", perm: 0o755, exec: true},
		{name: "first.test", content: "first", perm: 0o700, exec: true},
		{name: "data.bin", content: "data", perm: 0o644},
	}

	b := New()
	for _, f := range files {
		path := filepath.Join(dir, f.name)
		if err := os.WriteFile(path, []byte(f.content), f.perm); err != nil {
			t.Fatal(err)
		}
		// Do not depend on the umask.
		if err := os.Chmod(path, f.perm); err != nil {
			t.Fatal(err)
		}

		var err error
		if f.exec {
			err = b.Execute(path)
		} else {
			err = b.Embed(path)
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	var archive bytes.Buffer
	if err := b.Generate(&archive); err != nil {
		t.Fatal(err)
	}

	r := cpio.NewReader(&archive)

	// The generated executables come first.
	for _, name := range []string{"init", "bluebox-init"} {
		hdr, err := r.Next()
		if err != nil {
			t.Fatal(err)
		}
		if hdr.Name != name {
			t.Fatalf("got entry %q, want %q", hdr.Name, name)
		}
		if hdr.Size == 0 {
			t.Errorf("%s is empty", name)
		}
		// Windows does not keep the permission bits.
		if runtime.GOOS != "windows" && hdr.Mode.Perm()&0o100 == 0 {
			t.Errorf("%s is not executable: mode %v", name, hdr.Mode)
		}
	}

	for _, f := range files {
		hdr, err := r.Next()
		if err != nil {
			t.Fatal(err)
		}
		if hdr.Name != f.name {
			t.Fatalf("got entry %q, want %q", hdr.Name, f.name)
		}
		if !hdr.Mode.IsRegular() {
			t.Errorf("%s is not a regular file: %v", f.name, hdr.Mode)
		}
		// Windows does not keep the permission bits.
		if runtime.GOOS != "windows" && os.FileMode(hdr.Mode.Perm()) != f.perm {
			t.Errorf("%s has mode %o, want %o", f.name, hdr.Mode.Perm(), f.perm)
		}

		content, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != f.content {
			t.Errorf("%s has content %q, want %q", f.name, content, f.content)
		}
	}

	if _, err := r.Next(); err != io.EOF {
		t.Errorf("expected end of archive, got %v", err)
	}
}
