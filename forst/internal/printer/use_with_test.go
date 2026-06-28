package printer

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"github.com/sirupsen/logrus"
)

func printStmtString(stmt ast.Node) (string, error) {
	var p printer
	p.cfg = DefaultConfig()
	return p.printStmt(stmt)
}

func TestPrint_useNamedAndAnonymous(t *testing.T) {
	named := ast.UseNode{
		Ident:        &ast.Ident{ID: "logger"},
		ContractType: ast.TypeNode{Ident: "Logger"},
	}
	out, err := printStmtString(named)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "use logger: Logger") {
		t.Fatalf("named use print: %q", out)
	}

	anon := ast.UseNode{ContractType: ast.TypeNode{Ident: "Clock"}}
	out, err = printStmtString(anon)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "use Clock") {
		t.Fatalf("anonymous use print: %q", out)
	}
}

func TestPrint_methodContractShapeTypedef(t *testing.T) {
	shape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"info": {
				IsMethod: true,
				MethodParams: []ast.ParamNode{
					ast.SimpleParamNode{Ident: ast.Ident{ID: "msg"}, Type: ast.TypeNode{Ident: ast.TypeString}},
				},
			},
			"now": {
				IsMethod:          true,
				MethodParams:      nil,
				MethodReturnTypes: []ast.TypeNode{{Ident: ast.TypeInt}},
			},
		},
	}
	var p printer
	p.cfg = DefaultConfig()
	out, err := p.printShapeMultiline(shape, 1)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "info:") {
		t.Fatalf("method field should not use colon syntax: %q", out)
	}
	if !strings.Contains(out, "info(msg String)") {
		t.Fatalf("missing method signature: %q", out)
	}
	if !strings.Contains(out, "now(): Int") {
		t.Fatalf("missing return type method: %q", out)
	}
}

func TestPrint_withShapeLiteralWiring(t *testing.T) {
	with := ast.WithNode{
		Wiring: ast.ShapeNode{
			Fields: map[string]ast.ShapeFieldNode{
				"Logger": {Type: &ast.TypeNode{Ident: "NopLogger"}},
				"Clock":  {Type: &ast.TypeNode{Ident: "FakeClock"}},
			},
		},
		Body: []ast.Node{
			ast.UseNode{ContractType: ast.TypeNode{Ident: "Logger"}},
		},
	}
	out, err := printStmtString(with)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "with {") {
		t.Fatalf("missing with block: %q", out)
	}
	if !strings.Contains(out, "Logger:") || !strings.Contains(out, "Clock:") {
		t.Fatalf("missing wiring fields: %q", out)
	}
}

func TestPrint_withCompactWiringInsideFunction(t *testing.T) {
	fn := ast.FunctionNode{
		Ident: ast.Ident{ID: "TestHandle"},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{Ident: ast.Ident{ID: "t"}, Type: ast.TypeNode{Ident: "testing.T"}},
		},
		Body: []ast.Node{
			ast.WithNode{
				Wiring: ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"Logger": {Type: &ast.TypeNode{Ident: "NopLogger"}},
					},
				},
				Body: []ast.Node{
					ast.FunctionCallNode{
						Function: ast.Ident{ID: "Handle"},
						Arguments: []ast.ExpressionNode{
							ast.StringLiteralNode{Value: "tok-1"},
						},
					},
				},
			},
		},
	}
	out, err := Print(DefaultConfig(), []ast.Node{fn})
	if err != nil {
		t.Fatal(err)
	}
	want := "with {Logger: NopLogger"
	if !strings.Contains(out, want) {
		t.Fatalf("expected compact with wiring, got:\n%s", out)
	}
	if strings.Contains(out, "with {\n") {
		t.Fatalf("short wiring should stay on one line, got:\n%s", out)
	}
}

func TestFormatSource_withWiringCompact(t *testing.T) {
	t.Parallel()
	src := `package beta

func TestHandle(t *testing.T) {
	with {
		Logger: &NopLogger{},
	} {
		Handle("tok-1")
	}
}
`
	out, err := FormatSource(src, "t.ft", testLogger())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "with {\n") {
		t.Fatalf("expected compact with wiring, got:\n%s", out)
	}
	if !strings.Contains(out, "with { Logger: &NopLogger{}, } {") && !strings.Contains(out, "with {Logger: &NopLogger{}}") {
		t.Fatalf("missing compact with line, got:\n%s", out)
	}
}

func testLogger() *logrus.Logger {
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	return log
}

func TestPrint_withMultilineWiringWhenExceedsLineWidth(t *testing.T) {
	cfg := DefaultConfig()
	cfg.TypeDefLineWidth = 20
	with := ast.WithNode{
		Wiring: ast.ShapeNode{
			Fields: map[string]ast.ShapeFieldNode{
				"Logger": {Type: &ast.TypeNode{Ident: "NopLogger"}},
				"Clock":  {Type: &ast.TypeNode{Ident: "FakeClock"}},
			},
		},
		Body: []ast.Node{
			ast.UseNode{ContractType: ast.TypeNode{Ident: "Logger"}},
		},
	}
	var p printer
	p.cfg = cfg
	p.depth = 1
	out, err := p.printWith(with)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "with {\n") {
		t.Fatalf("expected broken wiring block, got:\n%s", out)
	}
	if !strings.Contains(out, "\t\tLogger:") && !strings.Contains(out, "\t\tClock:") {
		t.Fatalf("expected indented wiring fields, got:\n%s", out)
	}
	if !strings.Contains(out, "\n\t} {\n") {
		t.Fatalf("expected aligned wiring close before body, got:\n%s", out)
	}
}
