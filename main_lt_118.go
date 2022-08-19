//go:build !go1.18
// +build !go1.18

package main

import (
	"fmt"
)

func showVersion() {
	fmt.Printf("bluebox <unknown> from older Go version\n")
}
