package initramfs

import (
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
