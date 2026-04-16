package printer

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestFormatTypeDefNode_andPrintShapeOneLine(t *testing.T) {
	t.Parallel()
	typeDef := ast.TypeDefNode{
		Ident: "User",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	}
	out, err := FormatTypeDefNode(Config{Indent: "  "}, typeDef)
	if err != nil {
		t.Fatalf("FormatTypeDefNode: %v", err)
	}
	if !strings.Contains(out, "type User") {
		t.Fatalf("unexpected typedef output: %q", out)
	}
	p := &printer{cfg: Config{Indent: "  "}}
	shapeLine, err := p.printShapeOneLine(ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"enabled": {Type: &ast.TypeNode{Ident: ast.TypeBool}},
		},
	})
	if err != nil {
		t.Fatalf("printShapeOneLine: %v", err)
	}
	if !strings.Contains(shapeLine, "enabled") {
		t.Fatalf("unexpected shape line: %q", shapeLine)
	}
}

