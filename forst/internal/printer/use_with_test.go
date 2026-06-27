package printer

import (
	"strings"
	"testing"

	"forst/internal/ast"
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
