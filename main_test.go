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
			execs: []string{},
			args:  [][]string{},
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
		"without arguments": {
			input: `bazinga`,
			execs: []string{
				"bazinga",
			},
			args: [][]string{
				{},
			},
		},
	}

	for name, tc := range tests {
		name := name
		tc := tc
		t.Run(name, func(t *testing.T) {
			// Reset package global variables
			execs = []string{}
			args = [][]string{}

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

func TestEmbedEnvVar(t *testing.T) {
	tests := map[string]struct {
		input string
		env   map[string]string
	}{
		"no input": {
			input: "",
			env:   make(map[string]string),
		},
		"without value": {
			input: "key",
			env: map[string]string{
				"key": "TRUE",
			},
		},
		"key=value": {
			input: "key=value",
			env: map[string]string{
				"key": "value",
			},
		},
	}
	for name, tc := range tests {
		name := name
		tc := tc
		t.Run(name, func(t *testing.T) {
			// Reset package global variables
			for k := range env {
				delete(env, k)
			}

			if err := embedEnvVar(tc.input); err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if !reflect.DeepEqual(env, tc.env) {
				t.Fatalf("expected environment variables did not match. "+
					"Got: %#v\nExpected: %#v", env, tc.env)
			}

		})
	}
}
