package transformergo

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker"
)

func TestPipeline_pointerReturnStructLiteral(t *testing.T) {
	src := `package main

type Point = { x: Int, y: Int }

func origin(): *Point {
	p := Point{ x: 0, y: 1 }
	return &p
}

func main() {
	p := origin()
	println(string(p.x))
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{`type Point`, `*Point`, `return &`} {
		if !strings.Contains(out, sub) {
			t.Fatalf("missing %q in:\n%s", sub, out)
		}
	}
	assertGoParses(t, out)
}

func TestPipeline_functionParamStringAlias(t *testing.T) {
	src := `package main

type Tag = String

func echo(s: String): String {
	return s
}

func main() {
	println(echo("hi"))
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, `func echo(s string)`) {
		t.Fatalf("expected string param:\n%s", out)
	}
	assertGoParses(t, out)
}

func TestPipeline_destructuredParamInFunctionDecl(t *testing.T) {
	src := `package main

func pair({x, y}: { x: Int, y: Int }): Int {
	return x + y
}

func main() {
	println(string(pair({ x: 1, y: 2 })))
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, `func pair(x int, y int)`) {
		t.Fatalf("expected flattened destructured params:\n%s", out)
	}
	assertGoParses(t, out)
}

func TestPipeline_functionWithExplicitErrorReturn(t *testing.T) {
	src := `package main

func fail(): Error {
	return errors.New("boom")
}

func main() {}
`
	out := compileForstPipelineExt(t, src, pipelineOpts{goWorkspaceDir: moduleRootFromWD(t)})
	if !strings.Contains(out, `return errors.New`) {
		t.Fatalf("expected error return emit:\n%s", out)
	}
	assertGoParses(t, out)
}

func TestPipeline_nestedWithBlock(t *testing.T) {
	src := `package main

type Logger = { info(msg String) }
type Clock = { now(): Int }
type NopLogger = {}
type FakeClock = { fixedMs: Int }

func (NopLogger) info(msg String) {}
func (c FakeClock) now(): Int { return c.fixedMs }

func work() {
	use logger: Logger
	logger.info("ok")
}

func main() {
	with { Logger: NopLogger{} } {
		with { Clock: FakeClock{ fixedMs: 1 } } {
			work()
		}
	}
}
`
	out := compileForstPipelineExt(t, src, pipelineOpts{goWorkspaceDir: moduleRootFromWD(t)})
	if !strings.Contains(out, `Providers_`) {
		t.Fatalf("expected nested with providers emit:\n%s", out)
	}
	assertGoParses(t, out)
}

func TestPipeline_typedefUnionMembers(t *testing.T) {
	src := `package main

type ParseError = { message: String }
type IoError = { code: Int }
type AppError = ParseError | IoError

func main() {
	var e: AppError = ParseError{ message: "x" }
	println("ok")
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{`type ParseError`, `type IoError`, `type AppError`, `message`} {
		if !strings.Contains(out, sub) {
			t.Fatalf("missing %q in:\n%s", sub, out)
		}
	}
	assertGoParses(t, out)
}

func TestPipeline_mapAndSliceTypeFields(t *testing.T) {
	src := `package main

type Config = {
	labels: []String,
	scores: map[String]Int,
}

func main() {
	c := Config{ labels: ["a"], scores: map[String]Int{ "k": 1 } }
	println(string(len(c.labels)))
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{`[]string`, `map[string]int`, `type Config`} {
		if !strings.Contains(out, sub) {
			t.Fatalf("missing %q in:\n%s", sub, out)
		}
	}
	assertGoParses(t, out)
}

func TestPipeline_ensureBoolFalse(t *testing.T) {
	src := `package main

func main() {
	ok := false
	ensure ok is False()
	println("done")
}
`
	out := compileForstPipeline(t, src)
	if strings.Contains(out, `!ok`) {
		t.Fatalf("False() should not negate subject, got:\n%s", out)
	}
	assertGoParses(t, out)
}

func TestPipeline_stringAliasCoercionInPrintln(t *testing.T) {
	src := `package main

type Slug = String.Min(1)

func main() {
	s: Slug = "hello"
	println(s)
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, `type Slug`) {
		t.Fatalf("expected Slug typedef:\n%s", out)
	}
	assertGoParses(t, out)
}

func TestPipeline_shapeLiteralWithExplicitBaseType(t *testing.T) {
	src := `package main

type Point = { x: Int, y: Int }

func main() {
	p := Point{ x: 1, y: 2 }
	println(string(p.x))
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, `Point{`) {
		t.Fatalf("expected Point composite literal:\n%s", out)
	}
	assertGoParses(t, out)
}

func TestTransformType_mapChannelPointer(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	mapExpr, err := tr.transformType(ast.TypeNode{
		Ident:      ast.TypeMap,
		TypeParams: []ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeInt}},
	})
	if err != nil {
		t.Fatalf("map: %v", err)
	}
	if s := goExprString(t, mapExpr); !strings.Contains(s, "map[") {
		t.Fatalf("map type: %s", s)
	}

	sliceExpr, err := tr.transformType(ast.TypeNode{
		Ident:      ast.TypeArray,
		TypeParams: []ast.TypeNode{{Ident: ast.TypeString}},
	})
	if err != nil {
		t.Fatalf("slice: %v", err)
	}
	if s := goExprString(t, sliceExpr); !strings.Contains(s, "[]") {
		t.Fatalf("slice type: %s", s)
	}

	ptrExpr, err := tr.transformType(ast.TypeNode{
		Ident:      ast.TypePointer,
		TypeParams: []ast.TypeNode{{Ident: "Point"}},
	})
	if err != nil {
		t.Fatalf("pointer: %v", err)
	}
	if s := goExprString(t, ptrExpr); !strings.HasPrefix(s, "*") {
		t.Fatalf("pointer type: %s", s)
	}

	chanExpr, err := tr.transformType(ast.TypeNode{
		Ident:      ast.TypeChannel,
		TypeParams: []ast.TypeNode{{Ident: ast.TypeInt}},
	})
	if err != nil {
		t.Fatalf("channel: %v", err)
	}
	if s := goExprString(t, chanExpr); !strings.Contains(s, "chan") {
		t.Fatalf("channel type: %s", s)
	}
}

func TestTransformProviderSlotType_localContract(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tc.Defs["Logger"] = ast.TypeDefNode{
		Ident: "Logger",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"info": {IsMethod: true, MethodParams: []ast.ParamNode{
						ast.SimpleParamNode{Ident: ast.Ident{ID: "msg"}, Type: ast.TypeNode{Ident: ast.TypeString}},
					}},
				},
			},
		},
	}
	tr := setupTransformer(tc, log)
	slot := typechecker.ProviderSlot{
		RootIdent:    "Logger",
		ContractType: ast.TypeNode{Ident: "Logger"},
	}
	expr, err := tr.transformProviderSlotType(slot)
	if err != nil {
		t.Fatal(err)
	}
	if goExprString(t, expr) != "Logger" {
		t.Fatalf("got %s", goExprString(t, expr))
	}
}
