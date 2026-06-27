package ast

import "testing"

func TestShapeNode_IsMethodOnlyContract(t *testing.T) {
	methodOnly := ShapeNode{
		Fields: map[string]ShapeFieldNode{
			"info": {IsMethod: true},
		},
	}
	if !methodOnly.IsMethodOnlyContract() {
		t.Fatal("expected method-only")
	}
	mixed := ShapeNode{
		Fields: map[string]ShapeFieldNode{
			"x": {Type: &TypeNode{Ident: TypeString}},
			"m": {IsMethod: true},
		},
	}
	if mixed.IsMethodOnlyContract() {
		t.Fatal("mixed shape should not be method-only")
	}
}

func TestShapeFieldNode_ValueExpression(t *testing.T) {
	inner := ShapeNode{Fields: map[string]ShapeFieldNode{}}
	field := ShapeFieldNode{Shape: &inner}
	expr, ok := field.ValueExpression()
	if !ok {
		t.Fatal("expected expression from Shape field")
	}
	if _, ok := expr.(ShapeNode); !ok {
		t.Fatalf("got %T", expr)
	}
}

func TestIsUsablesWiringRoot(t *testing.T) {
	if !IsUsablesWiringRoot("main", nil) {
		t.Fatal("main is wiring root")
	}
	tParam := TypeNode{
		Ident:      TypePointer,
		TypeParams: []TypeNode{{Ident: "testing.T"}},
	}
	if !IsUsablesWiringRoot("TestFoo", []TypeNode{tParam}) {
		t.Fatal("Test* with *testing.T is wiring root")
	}
	if IsUsablesWiringRoot("outer", nil) {
		t.Fatal("outer is not wiring root")
	}
	if !IsUsablesWiringRoot("TestUnknown", nil) {
		t.Fatal("Test* without sig defaults to wiring root")
	}
}

func TestIsPublicExportIdent(t *testing.T) {
	if !IsPublicExportIdent("Handle") {
		t.Fatal("Handle is public")
	}
	if IsPublicExportIdent("handle") {
		t.Fatal("handle is not public")
	}
}

func TestIsTestingTParamType(t *testing.T) {
	ptr := TypeNode{
		Ident:      TypePointer,
		TypeParams: []TypeNode{{Ident: "testing.T"}},
	}
	if !IsTestingTParamType(ptr) {
		t.Fatal("expected *testing.T")
	}
	if IsTestingTParamType(TypeNode{Ident: "Logger"}) {
		t.Fatal("Logger is not testing.T")
	}
}
