package transformergo

import (
	"bytes"
	"go/format"
	"go/token"
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
	"forst/internal/typechecker"
)

// compileForstPipeline runs parse → typecheck → transform → go/format on Forst source.
func compileForstPipeline(t *testing.T, src string) string {
	t.Helper()
	log := ast.SetupTestLogger(nil)
	if !testing.Verbose() {
		log.SetOutput(bytes.NewBuffer(nil))
	}

	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	tc := typechecker.New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("typecheck: %v", err)
	}

	tr := New(tc, log)
	goFile, err := tr.TransformForstFileToGo(nodes)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, token.NewFileSet(), goFile); err != nil {
		t.Fatalf("go/format: %v", err)
	}
	return buf.String()
}

func TestPipeline_parse_typecheck_transform_goFormat(t *testing.T) {
	tests := []struct {
		name   string
		src    string
		needles []string // substrings that must appear in generated Go (stable signals)
	}{
		{
			name: "basic_function_and_return",
			src: `package main

func greet(): String {
	return "Hello"
}

func main() {
	println(greet())
}
`,
			needles: []string{`package main`, `func greet`, `return "Hello"`, `func main`},
		},
		{
			name: "import_fmt_and_call",
			src: `package main

import "fmt"

func main() {
	fmt.Println("ok")
}
`,
			needles: []string{`package main`, `"fmt"`, `fmt.Println`, `ok`},
		},
		{
			name: "type_def_and_struct_literal_return",
			src: `package main

type Point = { x: Int, y: Int }

func origin(): Point {
	return { x: 0, y: 1 }
}

func main() {
	p := origin()
	println("ok")
}
`,
			// Struct literal return + named shape type; avoid println(int): builtin println expects String.
			needles: []string{`type Point`, `func origin`, `return`, `struct`},
		},
		{
			name: "type_guard_and_ensure_block",
			src: `package main

type Password = String

is (password Password) Strong {
	ensure password is Min(12)
}

func main() {
	password: Password = "1234567890123"
	ensure password is Strong() {
		println("weak")
	}
	println("done")
}
`,
			// Guard implementations are emitted as func G_<hash>(...) bool; ensure block uses os.Exit on failure.
			needles: []string{`package main`, `func G_`, `Password`, `os.Exit`, `func main`},
		},
		{
			name: "arithmetic_int_return",
			src: `package main

func sum(): Int {
	return 3 + 4
}

func main() {
	println("ok")
}
`,
			// Binary `+` is inferred in sum; main stays string-only for println.
			needles: []string{`func sum`, `+`, `return`, `func main`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := compileForstPipeline(t, tt.src)
			for _, sub := range tt.needles {
				if !strings.Contains(out, sub) {
					t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
				}
			}
		})
	}
}

func TestPipeline_emitted_go_is_gofmt_clean(t *testing.T) {
	src := `package main

func main() {
	println("x")
}
`
	log := ast.SetupTestLogger(nil)
	if !testing.Verbose() {
		log.SetOutput(bytes.NewBuffer(nil))
	}
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := typechecker.New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	tr := New(tc, log)
	goFile, err := tr.TransformForstFileToGo(nodes)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, token.NewFileSet(), goFile); err != nil {
		t.Fatal(err)
	}
	formatted := buf.Bytes()
	// Second pass through gofmt must be a no-op on valid output.
	again, err := format.Source(formatted)
	if err != nil {
		t.Fatalf("format.Source: %v\n%s", err, formatted)
	}
	if !bytes.Equal(formatted, again) {
		t.Fatalf("emitted Go not stable under gofmt")
	}
}
