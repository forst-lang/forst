package ast

import "testing"

func TestOkErrExpr_StringKindAndTypePredicates(t *testing.T) {
	t.Parallel()
	okNode := OkExprNode{Value: IntLiteralNode{Value: 1}}
	if okNode.Kind() != NodeKindOkExpr {
		t.Fatalf("Ok kind mismatch: %v", okNode.Kind())
	}
	if okNode.String() == "" {
		t.Fatal("expected Ok string")
	}
	errNode := ErrExprNode{Value: StringLiteralNode{Value: "e"}}
	if errNode.Kind() != NodeKindErrExpr {
		t.Fatalf("Err kind mismatch: %v", errNode.Kind())
	}
	if errNode.String() == "" {
		t.Fatal("expected Err string")
	}
	resultType := TypeNode{Ident: TypeResult, TypeParams: []TypeNode{{Ident: TypeInt}, {Ident: TypeError}}}
	if !resultType.IsResultType() {
		t.Fatal("expected result type")
	}
	unionType := TypeNode{Ident: TypeUnion, TypeParams: []TypeNode{{Ident: TypeInt}, {Ident: TypeString}}}
	if !unionType.IsUnionType() {
		t.Fatal("expected union type")
	}
	intersectionType := TypeNode{Ident: TypeIntersection, TypeParams: []TypeNode{{Ident: TypeInt}, {Ident: TypeString}}}
	if !intersectionType.IsIntersectionType() {
		t.Fatal("expected intersection type")
	}
}

