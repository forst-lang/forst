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

func TestFormatTypeDefNode_nestedUnionFlattens(t *testing.T) {
	t.Parallel()
	expr := ast.TypeDefBinaryExpr{
		Op: ast.TokenBitwiseOr,
		Left: ast.TypeDefBinaryExpr{
			Op:    ast.TokenBitwiseOr,
			Left:  ast.TypeDefShapeExpr{Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{"a": {Type: &ast.TypeNode{Ident: ast.TypeInt}}}}},
			Right: ast.TypeDefShapeExpr{Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{"b": {Type: &ast.TypeNode{Ident: ast.TypeInt}}}}},
		},
		Right: ast.TypeDefShapeExpr{Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{"c": {Type: &ast.TypeNode{Ident: ast.TypeInt}}}}},
	}
	typeDef := ast.TypeDefNode{Ident: "U", Expr: expr}
	out, err := FormatTypeDefNode(Config{Indent: "  "}, typeDef)
	if err != nil {
		t.Fatalf("FormatTypeDefNode: %v", err)
	}
	if !strings.Contains(out, "type U") || !strings.Contains(out, "|") {
		t.Fatalf("unexpected typedef union output: %q", out)
	}
}

