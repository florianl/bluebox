package initramfs

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"testing"
)

// hostile holds values that would break out of a string literal if they were not quoted.
var hostile = []string{
	`"); os.Exit(1); fmt.Println("`,
	"`+os.Args[0]+`",
	`back\slash`,
	"new\nline",
	"{{.}} %s %d",
	"",
}

// stringLiterals parses src as Go source and returns the values of the string literals in the
// declaration of the package level variable name.
func stringLiterals(t *testing.T, src []byte, name string) []string {
	t.Helper()

	f, err := parser.ParseFile(token.NewFileSet(), "", src, 0)
	if err != nil {
		t.Fatalf("generated source does not parse: %v\n%s", err, src)
	}

	values := []string{}
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) != 1 || vs.Names[0].Name != name {
				continue
			}
			ast.Inspect(vs, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					v, err := strconv.Unquote(lit.Value)
					if err != nil {
						t.Fatal(err)
					}
					values = append(values, v)
				}
				return true
			})
		}
	}

	return values
}

func TestRenderBlueboxQuotesInput(t *testing.T) {
	b := New()
	// Execute requires existing files, so set the fields directly. Only the base name of the path
	// ends up in the source.
	for _, h := range hostile {
		b.execs = append(b.execs, execEntry{path: h, args: hostile})
		b.Setenv(h, h)
	}

	var src bytes.Buffer
	if err := b.renderBluebox(&src); err != nil {
		t.Fatal(err)
	}

	var wantArgs, wantEnv []string
	for _, h := range hostile {
		wantArgs = append(wantArgs, hostile...)
		wantEnv = append(wantEnv, h, h)
	}

	if got := stringLiterals(t, src.Bytes(), "exeArg"); !reflect.DeepEqual(got, wantArgs) {
		t.Errorf("exeArg = %q, want %q", got, wantArgs)
	}
	if got := stringLiterals(t, src.Bytes(), "envVars"); !reflect.DeepEqual(got, wantEnv) {
		t.Errorf("envVars = %q, want %q", got, wantEnv)
	}
	if got := stringLiterals(t, src.Bytes(), "execs"); len(got) != len(hostile) {
		t.Errorf("execs has %d entries, want %d: %q", len(got), len(hostile), got)
	}
}

func TestRenderInitQuotesInput(t *testing.T) {
	var env []environment
	for _, h := range hostile {
		env = append(env,
			mount{source: h, target: h, fstype: h, data: h, targetCreate: true, targetPerm: 0o755},
			mount{source: h, target: h, fstype: h, data: h},
			nod{path: h},
		)
	}

	var hostileSrc bytes.Buffer
	if err := renderInit(&hostileSrc, env); err != nil {
		t.Fatal(err)
	}

	var plainSrc bytes.Buffer
	if err := renderInit(&plainSrc, nil); err != nil {
		t.Fatal(err)
	}

	// Hostile values must only add statements to main and nothing else, like new calls.
	count := func(src []byte) (calls, decls int) {
		f, err := parser.ParseFile(token.NewFileSet(), "", src, 0)
		if err != nil {
			t.Fatalf("generated source does not parse: %v\n%s", err, src)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if _, ok := n.(*ast.CallExpr); ok {
				calls++
			}
			return true
		})
		return calls, len(f.Decls)
	}

	plainCalls, plainDecls := count(plainSrc.Bytes())
	hostileCalls, hostileDecls := count(hostileSrc.Bytes())

	// Per hostile value: mount with target creation has 6 calls, plain mount 4 and nod 5,
	// counting the uintptr conversion.
	if want := plainCalls + len(hostile)*15; hostileCalls != want {
		t.Errorf("generated init has %d calls, want %d", hostileCalls, want)
	}
	if hostileDecls != plainDecls {
		t.Errorf("generated init has %d declarations, want %d", hostileDecls, plainDecls)
	}
}

func TestRenderInitDefault(t *testing.T) {
	var src bytes.Buffer
	if err := renderInit(&src, defaultEnvironment); err != nil {
		t.Fatal(err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "", src.Bytes(), 0); err != nil {
		t.Fatalf("generated source does not parse: %v", err)
	}
}
