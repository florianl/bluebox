//go:build go1.18
// +build go1.18

package main

import (
	"fmt"
	"runtime/debug"
)

func showVersion() {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Printf("bluebox@unknown\n")
		return
	}

	var rev string

	for _, bs := range bi.Settings {
		if bs.Key == "vcs.revision" {
			rev = bs.Value
			break
		}
	}
	fmt.Printf("bluebox@%s\n", rev)
}
