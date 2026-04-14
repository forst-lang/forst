package ast

import "testing"

func TestOkErrExprNode_StringAndKind(t *testing.T) {
	okNode := OkExprNode{Value: StringLiteralNode{Value: "x"}}
	if okNode.Kind() != NodeKindOkExpr {
		t.Fatalf("ok kind: %v", okNode.Kind())
	}
	if got := okNode.String(); got != `Ok("x")` {
		t.Fatalf("ok string: %q", got)
	}

	errNode := ErrExprNode{Value: StringLiteralNode{Value: "boom"}}
	if errNode.Kind() != NodeKindErrExpr {
		t.Fatalf("err kind: %v", errNode.Kind())
	}
	if got := errNode.String(); got != `Err("boom")` {
		t.Fatalf("err string: %q", got)
	}
}

func TestOkErrExprNode_EmptyStringForms(t *testing.T) {
	if got := (OkExprNode{}).String(); got != "Ok()" {
		t.Fatalf("ok empty: %q", got)
	}
	if got := (ErrExprNode{}).String(); got != "Err()" {
		t.Fatalf("err empty: %q", got)
	}
}

func TestTypeNode_ResultTupleUnionIntersectionPredicates(t *testing.T) {
	if !(TypeNode{
		Ident:      TypeResult,
		TypeParams: []TypeNode{{Ident: TypeString}, {Ident: TypeError}},
	}.IsResultType()) {
		t.Fatal("expected IsResultType true")
	}
	if !(TypeNode{
		Ident:      TypeTuple,
		TypeParams: []TypeNode{{Ident: TypeString}},
	}.IsTupleType()) {
		t.Fatal("expected IsTupleType true")
	}
	if !(TypeNode{
		Ident:      TypeUnion,
		TypeParams: []TypeNode{{Ident: TypeString}, {Ident: TypeInt}},
	}.IsUnionType()) {
		t.Fatal("expected IsUnionType true")
	}
	if !(TypeNode{
		Ident:      TypeIntersection,
		TypeParams: []TypeNode{{Ident: TypeString}, {Ident: TypeInt}},
	}.IsIntersectionType()) {
		t.Fatal("expected IsIntersectionType true")
	}
}
