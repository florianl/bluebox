package main

import (
	"reflect"
	"testing"
)

func TestParseAdditionalExecs(t *testing.T) {
	tests := map[string]struct {
		input []string
		execs []string
		args  [][]string
	}{
		"no input": {
			input: []string{},
		},
		"single input": {
			input: []string{`foo:"ba zinga"`},
			execs: []string{
				"foo",
			},
			args: [][]string{
				{"ba", "zinga"},
			},
		},
		"multiple inputs": {
			input: []string{
				`foo:bar`,
				`go:"-123 -456 -789"`,
				`date`,
			},
			execs: []string{
				"foo",
				"go",
				"date",
			},
			args: [][]string{
				{"bar"},
				{"-123", "-456", "-789"},
				{},
			},
		},
	}

	for name, tc := range tests {
		name := name
		tc := tc
		t.Run(name, func(t *testing.T) {
			execs, args, err := parseAdditionalExecs(tc.input)
			if err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}
			if !reflect.DeepEqual(execs, tc.execs) {
				t.Fatalf("expected executables did not match. "+
					"Got: %#v\nExpected: %#v", args, tc.args)
			}
			if !reflect.DeepEqual(args, tc.args) {
				t.Fatalf("expected arguments did not match. "+
					"Got: %#v\nExpected: %#v", args, tc.args)
			}
		})
	}
}
