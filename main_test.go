package main

import (
	"reflect"
	"testing"
)

func TestEmbedExec(t *testing.T) {
	tests := map[string]struct {
		input string
		execs []string
		args  [][]string
	}{
		"no input": {
			input: "",
		},
		"with argument input": {
			input: `foo:"bar"`,
			execs: []string{
				"foo",
			},
			args: [][]string{
				{"bar"},
			},
		},
		"with multiple argument inputs": {
			input: `go:"-123 -456 -789"`,
			execs: []string{
				"go",
			},
			args: [][]string{
				{"-123", "-456", "-789"},
			},
		},
	}

	for name, tc := range tests {
		name := name
		tc := tc
		t.Run(name, func(t *testing.T) {
			// Reset package global variables
			execs = execs[:0]
			args = args[:0]

			if err := embedExec(tc.input); err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}
			if !reflect.DeepEqual(execs, tc.execs) {
				t.Fatalf("expected executables did not match. "+
					"Got: %#v\nExpected: %#v", execs, tc.execs)
			}
			if !reflect.DeepEqual(args, tc.args) {
				t.Fatalf("expected arguments did not match. "+
					"Got: %#v\nExpected: %#v", args, tc.args)
			}
		})
	}
}
