package ast

import "testing"

func TestFunctionNode_helpers(t *testing.T) {
	fn := FunctionNode{
		Ident:       Ident{ID: "main"},
		ReturnTypes: []TypeNode{{Ident: TypeVoid}},
		Params:      []ParamNode{},
		Body:        []Node{},
	}
	if fn.Kind() != NodeKindFunction {
		t.Fatal(fn.Kind())
	}
	if fn.String() != "Function(main)" || fn.GetIdent() != "main" {
		t.Fatalf("String/GetIdent: %q %q", fn.String(), fn.GetIdent())
	}
	if !fn.HasMainFunctionName() {
		t.Fatal("expected main")
	}
	if !fn.HasExplicitReturnType() {
		t.Fatal("expected explicit return")
	}
	other := FunctionNode{Ident: Ident{ID: "Other"}, ReturnTypes: []TypeNode{}, Body: []Node{}}
	if other.String() != "Function(Other)" || other.GetIdent() != "Other" {
		t.Fatal(other.String(), other.GetIdent())
	}
	if other.HasMainFunctionName() {
		t.Fatal("not main")
	}
	if other.HasExplicitReturnType() {
		t.Fatal("no return type")
	}
}
