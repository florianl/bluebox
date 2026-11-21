package main

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
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
		"empty arguments": {
			input: `foo:`,
			execs: []string{"foo"},
			args:  [][]string{{}},
		},
		"repeated white space": {
			input: `foo:"-a   -b "`,
			execs: []string{"foo"},
			args:  [][]string{{"-a", "-b"}},
		},
		"quoted argument": {
			input: `foo:"-run 'Test A' -v"`,
			execs: []string{"foo"},
			args:  [][]string{{"-run", "Test A", "-v"}},
		},
		"multiple double quoted arguments": {
			input: `foo:"a b" "c d"`,
			execs: []string{"foo"},
			args:  [][]string{{"a b", "c d"}},
		},
		"escaped white space": {
			input: `foo:a\ b ''`,
			execs: []string{"foo"},
			args:  [][]string{{"a b", ""}},
		},
		"colon in argument": {
			input: `foo:-test.run=a:b`,
			execs: []string{"foo"},
			args:  [][]string{{"-test.run=a:b"}},
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

func TestSplitExecColonInPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("colons are not allowed in file names")
	}

	path := filepath.Join(t.TempDir(), "a:b.test")
	if err := os.WriteFile(path, nil, 0o755); err != nil {
		t.Fatal(err)
	}

	tests := map[string][2]string{
		path:               {path, ""},
		path + ":-test.v":  {path, "-test.v"},
		path + ":-run=x:y": {path, "-run=x:y"},
		"missing:-run=x:y": {"missing", "-run=x:y"},
		"missing":          {"missing", ""},
	}

	for input, want := range tests {
		cmd, options := splitExec(input)
		if cmd != want[0] || options != want[1] {
			t.Errorf("splitExec(%q) = %q, %q, want %q, %q", input, cmd, options, want[0], want[1])
		}
	}
}
