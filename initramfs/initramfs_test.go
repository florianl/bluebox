package initramfs

import (
	"io"
	"testing"
)

func TestBluebox(t *testing.T) {
	b := New()
	if err := b.Generate(io.Discard); err != nil {
		t.Fatal(err)
	}
}
