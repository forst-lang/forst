package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParseShapeType_ArrayInlineShapeField(t *testing.T) {
	t.Parallel()
	input := `{ items: Array({ id: String, quantity: Int }) }`
	p := NewTestParser(input, ast.SetupTestLogger(nil))
	shape := p.parseShapeType()
	field := shape.Fields["items"]
	if field.Type == nil {
		t.Fatal("items field has no type")
	}
	if field.Type.Ident != ast.TypeArray {
		t.Fatalf("expected Array type, got %s", field.Type.Ident)
	}
	if len(field.Type.TypeParams) != 1 {
		t.Fatalf("expected 1 type param, got %d", len(field.Type.TypeParams))
	}
	elem := field.Type.TypeParams[0]
	if elem.Assertion == nil || len(elem.Assertion.Constraints) == 0 {
		t.Fatalf("expected assertion on array element type, got %+v", elem)
	}
	nested := elem.Assertion.Constraints[0].Args[0].Shape
	if nested == nil || len(nested.Fields) != 2 {
		t.Fatalf("expected nested shape with 2 fields, got %+v", nested)
	}
}
