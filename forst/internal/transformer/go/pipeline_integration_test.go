package transformergo

import (
	"bytes"
	"go/format"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
	"forst/internal/typechecker"
)

// pipelineOpts configures compileForstPipelineExt (Go workspace / optional skip for FFI tests).
type pipelineOpts struct {
	goWorkspaceDir     string
	skipUnlessGoImport string // if set, t.Skip when go/packages did not load this import local name
}

// compileForstPipeline runs parse → typecheck → transform → go/format on Forst source.
func compileForstPipeline(t *testing.T, src string) string {
	t.Helper()
	return compileForstPipelineExt(t, src, pipelineOpts{})
}

// compileForstPipelineExt runs parse → typecheck → transform → go/format with optional GoWorkspaceDir.
func compileForstPipelineExt(t *testing.T, src string, opts pipelineOpts) string {
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
	tc.GoWorkspaceDir = opts.goWorkspaceDir
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	if opts.skipUnlessGoImport != "" && !tc.GoImportPackageLoaded(opts.skipUnlessGoImport) {
		t.Skipf("%s not loaded (go/packages or workspace)", opts.skipUnlessGoImport)
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

func moduleRootFromWD(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found from cwd")
		}
		dir = parent
	}
}

func TestPipeline_parse_typecheck_transform_goFormat(t *testing.T) {
	tests := []struct {
		name    string
		src     string
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
		{
			name: "if_else_branch",
			src: `package main

func main() {
	n := 1
	if n > 0 {
		println("yes")
	} else {
		println("no")
	}
}
`,
			needles: []string{`if `, `else`, `println`, `func main`},
		},
		{
			name: "if_else_if_else_chain",
			src: `package main

func main() {
	n := 2
	if n > 10 {
		println("a")
	} else if n < 0 {
		println("b")
	} else {
		println("c")
	}
}
`,
			// Emitter lowers else-if to nested Go if statements.
			needles: []string{`if `, `else`, `println("c")`, `func main`},
		},
		{
			name: "if_with_short_decl_init",
			src: `package main

func main() {
	if x := 1; x > 0 {
		println("ok")
	}
}
`,
			needles: []string{`if `, `x :=`, `println`, `func main`},
		},
		{
			name: "for_range_over_slice",
			src: `package main

func main() {
	xs := [1, 2]
	for range xs {
		println("r")
	}
}
`,
			needles: []string{`for `, `range`, `println`, `func main`},
		},
		{
			name: "defer_and_go_statements",
			src: `package main

func work() {}

func main() {
	defer work()
	go work()
	println("ok")
}
`,
			needles: []string{`defer work()`, `go work()`, `func main`},
		},
		{
			name: "builtin_len_string",
			src: `package main

func main() {
	println(len("hi"))
}
`,
			needles: []string{`package main`, `len("hi")`, `println`},
		},
		{
			name: "builtin_len_array_literal",
			src: `package main

func main() {
	println(len([1, 2, 3]))
}
`,
			needles: []string{`len(`, `func main`},
		},
		{
			name: "builtin_min_max_literals",
			src: `package main

func main() {
	println(min(1, 2, 3))
	println(max(3, 4))
}
`,
			needles: []string{`min(`, `max(`, `func main`},
		},
		{
			name: "slice_shape_field_and_param_emit_go_slice",
			src: `package main

type Row = { cells: []String }

func getCell(cells []String, idx Int): String {
	return cells[idx]
}

func main() {
	println(getCell(["x"], 0))
}
`,
			// Shape fields and parameters must use Go []string, not a hash-only type name (would break indexing).
			needles: []string{`type Row`, `[]string`, `cells []string`, `cells[idx]`},
		},
		{
			name: "return_user_defined_struct_type_via_call_expr",
			src: `package main

type R = { ok: Bool }

func inner(): R {
	return { ok: true }
}

func outer(): R {
	return inner()
}

func main() {
	x := outer()
	println(x.ok)
}
`,
			needles: []string{`func inner`, `func outer`, `return inner()`, `type R`},
		},
		{
			name: "ensure_greater_than_negative_int_literal_with_or",
			src: `package main

import "errors"

func bad(msg String): Error {
	return errors.New(msg)
}

func f(row Int): Result(String, Error) {
	ensure row is GreaterThan(-1) or bad("x")
	return Ok("ok")
}

func main() {
	println("hi")
}
`,
			needles: []string{`-1`, `row`, `errors`},
		},
		{
			name: "return_multi_value_call_single_expr_not_padded_with_nil",
			src: `package main

func inner(): Result(Int, Error) {
	return Ok(1)
}

func outer(): Result(Int, Error) {
	return inner()
}

func main() {
	x := outer()
	println(x)
}
`,
			needles: []string{`return inner()`, `func outer`},
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

// TestEmitValidation_* cases assert generated Go for built-in constraints and type guards (grep-friendly; see internal/coveragehotspots).
func TestPipeline_return_multi_value_call_not_padded_with_nil(t *testing.T) {
	src := `package main

func inner(): Result(Int, Error) {
	return Ok(1)
}

func outer(): Result(Int, Error) {
	return inner()
}

func main() {
	x := outer()
	println(x)
}
`
	out := compileForstPipeline(t, src)
	if strings.Contains(out, "inner(), nil") {
		t.Fatalf("multi-value return must not be padded (invalid Go); got:\n%s", out)
	}
	if !strings.Contains(out, "return inner()") {
		t.Fatalf("expected `return inner()` in:\n%s", out)
	}
}

func TestEmitValidation_builtinMinOnString(t *testing.T) {
	src := `package main

func checkLen(name String) {
	ensure name is Min(1)
}

func main() {
	checkLen("hi")
	println("ok")
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{`func checkLen`, `len(`, `String.Min(1)`, `errors.New`, `package main`} {
		if !strings.Contains(out, sub) {
			t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
		}
	}
}

func TestEmitValidation_builtinLessThanOnInt(t *testing.T) {
	src := `package main

func capSpeed(speed Int) {
	ensure speed is LessThan(100)
}

func main() {
	capSpeed(50)
	println("ok")
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{`func capSpeed`, `100`, `package main`} {
		if !strings.Contains(out, sub) {
			t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
		}
	}
}

func TestEmitValidation_typeGuardStrongPassword(t *testing.T) {
	src := `package main

type Password = String

is (password Password) Strong {
	ensure password is Min(12)
}

func main() {
	password: Password = "1234567890123"
	ensure password is Strong() {
		println("weak")
	}
	println("ok")
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{`func G_`, `len(password)`, `Password`, `os.Exit`, `package main`} {
		if !strings.Contains(out, sub) {
			t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
		}
	}
}

func TestPipeline_singleAssignResultCall_emitsTwoValueAssignAndPrintln(t *testing.T) {
	src := `package main

func f(): Result(Int, Error) {
	return Ok(1)
}

func main() {
	x := f()
	println(x)
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{
		`x, xErr :=`,
		`println(x, xErr)`,
		`func f()`,
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
		}
	}
}

func TestPipeline_discardedResultCallStmt_emitsBareCallExprStmt(t *testing.T) {
	src := `package main

func f(): Result(Int, Error) {
	return Ok(1)
}

func main() {
	f()
}
`
	out := compileForstPipeline(t, src)
	if strings.Contains(out, `_, _ = f()`) {
		t.Fatalf("did not expect blank assignment; Go allows discarding multi-return via expression statement, got:\n%s", out)
	}
	if !strings.Contains(out, "main() {\n\tf()\n") {
		t.Fatalf("expected bare f() call statement in main, got:\n%s", out)
	}
}

func TestPipeline_discardedStrconvAtoiStmt_emitsBareCallExprStmt(t *testing.T) {
	dir := moduleRootFromWD(t)
	src := `package main

import "strconv"

func main() {
	strconv.Atoi("42")
}
`
	out := compileForstPipelineExt(t, src, pipelineOpts{
		goWorkspaceDir:     dir,
		skipUnlessGoImport: "strconv",
	})
	if strings.Contains(out, `_, _ = strconv.Atoi`) {
		t.Fatalf("did not expect blank assignment; Go allows discarding multi-return via expression statement, got:\n%s", out)
	}
	if !strings.Contains(out, "main() {\n\tstrconv.Atoi(\"42\")\n") {
		t.Fatalf("expected bare strconv.Atoi call statement in main, got:\n%s", out)
	}
}

func TestPipeline_ifResultIsOk_emitsErrNilCheck(t *testing.T) {
	src := `package main

func f(): Result(Int, Error) {
	return Ok(1)
}

func main() {
	x := f()
	if x is Ok() {
		println(x)
	}
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, `xErr == nil`) {
		t.Fatalf("expected `if` condition to check success error is nil, got:\n%s", out)
	}
}

func TestPipeline_shapeFieldResult_IntError_emitsStructStorageAndSelectors(t *testing.T) {
	src := `package main

type Wrap = {
	r: Result(Int, Error),
}

func okInt(): Result(Int, Error) {
	return Ok(42)
}

func main() {
	x := okInt()
	w := { r: x }
	if w.r is Ok() {
		println(w.r)
	}
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, "Err error") || !strings.Contains(out, "V") {
		t.Fatalf("expected Wrap field to lower Result to struct { V …; Err error }, got:\n%s", out)
	}
	if !strings.Contains(out, "w.r.Err == nil") {
		t.Fatalf("expected if w.r is Ok() to check w.r.Err == nil, got:\n%s", out)
	}
	if !strings.Contains(out, "println(w.r.V)") {
		t.Fatalf("expected println narrowed field to use w.r.V, got:\n%s", out)
	}
}

func TestPipeline_shapeFieldResult_println_unnarrowed_expandsVAndErr(t *testing.T) {
	src := `package main

type Wrap = {
	r: Result(Int, Error),
}

func okInt(): Result(Int, Error) {
	return Ok(42)
}

func main() {
	x := okInt()
	w := { r: x }
	println(w.r)
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, "println(w.r.V, w.r.Err)") {
		t.Fatalf("expected println on unnarrowed struct Result field to expand to V and Err, got:\n%s", out)
	}
}

func TestPipeline_shapeFieldResult_ensureOk_emitsCompoundErrCheck(t *testing.T) {
	src := `package main

type Wrap = {
	r: Result(Int, Error),
}

func okInt(): Result(Int, Error) {
	return Ok(42)
}

func main() {
	x := okInt()
	w := { r: x }
	ensure w.r is Ok()
	println(w.r)
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, `!(w.r.Err == nil)`) && !strings.Contains(out, `w.r.Err != nil`) {
		t.Fatalf("expected ensure w.r is Ok() to branch on w.r.Err, got:\n%s", out)
	}
}

func TestPipeline_ensure_string_min_infersBaseType(t *testing.T) {
	src := `package main

func main() {
	s := "ab"
	ensure s is Min(1)
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, "len(s)") {
		t.Fatalf("expected ensure Min on string to use len(s), got:\n%s", out)
	}
}

func TestPipeline_ensure_failure_on_Result_struct_returns_two_values_not_nil(t *testing.T) {
	// Regression: Result(S, Error) is one Forst return type; ensure failure must lower to
	// (zero S, error), not a single nil (invalid for struct S) or one return value.
	src := `package main

type Payload = { n: Int }

func f(): Result(Payload, Error) {
	x := 0
	ensure x is GreaterThan(0)
	return Ok({ n: 1 })
}

func main() {}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, "Payload{") {
		t.Fatalf("expected zero composite literal for struct success type on ensure failure, got:\n%s", out)
	}
	if !strings.Contains(out, "errors.New") {
		t.Fatalf("expected errors.New on ensure failure for Result return, got:\n%s", out)
	}
	// Old bug: transformType(Result) failed and produced a lone `return nil`
	if strings.Contains(out, "return nil") {
		t.Fatalf("ensure failure must not emit bare return nil for Result(Payload, Error), got:\n%s", out)
	}
}

func TestPipeline_ensureResultIsOk_emitsErrNilCheck(t *testing.T) {
	src := `package main

func f(): Result(Int, Error) {
	return Ok(1)
}

func main() {
	x := f()
	ensure x is Ok()
	println(x)
}
`
	out := compileForstPipeline(t, src)
	// Ensure runs the error path when the predicate fails: !(xErr == nil) (equivalent to xErr != nil)
	if !strings.Contains(out, `!(xErr == nil)`) && !strings.Contains(out, `xErr != nil`) {
		t.Fatalf("expected ensure failure branch on non-Ok result (negated xErr == nil), got:\n%s", out)
	}
}

func TestPipeline_fmtPrintln_resultSplitExpandsLikePrintln(t *testing.T) {
	src := `package main

import "fmt"

func f(): Result(Int, Error) {
	return Ok(1)
}

func main() {
	x := f()
	fmt.Println(x)
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{
		`x, xErr :=`,
		`fmt.Println(x, xErr)`,
		`import "fmt"`,
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
		}
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
