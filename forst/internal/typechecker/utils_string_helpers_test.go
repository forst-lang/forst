package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestFormatTypeList_andTypecheckErrors(t *testing.T) {
	t.Parallel()
	if got := formatTypeList(nil); got != "()" {
		t.Fatalf("empty formatTypeList: %q", got)
	}
	types := []ast.TypeNode{{Ident: ast.TypeInt}, {Ident: ast.TypeString}}
	if got := formatTypeList(types); got == "" || !strings.Contains(got, "Int") {
		t.Fatalf("unexpected formatTypeList: %q", got)
	}
	msg := typecheckError("boom")
	if !strings.Contains(msg, "Typecheck error") || !strings.Contains(msg, "boom") {
		t.Fatalf("unexpected typecheckError: %q", msg)
	}
	node := ast.Node(ast.VariableNode{Ident: ast.Ident{ID: "x"}})
	withNode := typecheckErrorMessageWithNode(&node, "bad")
	if !strings.Contains(withNode, "Typecheck error at") || !strings.Contains(withNode, "bad") {
		t.Fatalf("unexpected typecheckErrorMessageWithNode: %q", withNode)
	}
}

func TestFailWithTypeMismatch_containsContext(t *testing.T) {
	t.Parallel()
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}}
	err := failWithTypeMismatch(fn, []ast.TypeNode{{Ident: ast.TypeInt}}, []ast.TypeNode{{Ident: ast.TypeString}}, "prefix")
	if err == nil {
		t.Fatal("expected mismatch error")
	}
	if !strings.Contains(err.Error(), "prefix") || !strings.Contains(err.Error(), "function f") {
		t.Fatalf("unexpected mismatch error: %v", err)
	}
}

func TestIsShapeSuperset_andRegisterHashBasedType(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	fields := map[string]ast.ShapeFieldNode{
		"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
	}
	RegisterHashBasedType(tc, "T_testhash", fields)
	if _, ok := tc.Defs["T_testhash"]; !ok {
		t.Fatal("expected registered hash-based type in defs")
	}
	shapeA := ast.ShapeNode{Fields: fields}
	shapeB := ast.ShapeNode{Fields: fields}
	if !tc.IsShapeSuperset(shapeA, shapeB) {
		t.Fatal("expected identical shapes to be superset-compatible")
	}
}

