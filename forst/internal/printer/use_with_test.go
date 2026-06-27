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
