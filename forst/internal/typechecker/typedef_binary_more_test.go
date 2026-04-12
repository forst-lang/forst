package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestFlattenUnionForCombine_nestedUnionMember(t *testing.T) {
	t.Parallel()
	a := ast.TypeNode{Ident: ast.TypeIdent("A")}
	b := ast.TypeNode{Ident: ast.TypeIdent("B")}
	c := ast.TypeNode{Ident: ast.TypeIdent("C")}
	inner := ast.TypeNode{Ident: ast.TypeUnion, TypeParams: []ast.TypeNode{a, b}, TypeKind: ast.TypeKindBuiltin}
	outer := ast.TypeNode{Ident: ast.TypeUnion, TypeParams: []ast.TypeNode{inner, c}, TypeKind: ast.TypeKindBuiltin}
	got := flattenUnionForCombine(outer)
	if len(got) != 3 {
		t.Fatalf("len=%d want 3: %#v", len(got), got)
	}
}

func TestTypeDefExprToTypeNode_shapeExpr(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	u, err := tc.TypeDefExprToTypeNode(ast.TypeDefShapeExpr{
		Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if u.Ident == "" || u.TypeKind != ast.TypeKindHashBased {
		t.Fatalf("expected hash-based type node, got %+v", u)
	}
}

func TestTypeDefExprToTypeNode_errorExprErrors(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	_, err := tc.TypeDefExprToTypeNode(ast.TypeDefErrorExpr{
		Payload: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}},
	})
	if err == nil {
		t.Fatal("expected error for anonymous TypeDefErrorExpr in type expression")
	}
}

func TestTypeDefExprToTypeNode_binaryPropagatesLeftError(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	aName := ast.TypeIdent("A")
	_, err := tc.TypeDefExprToTypeNode(ast.TypeDefBinaryExpr{
		Left:  ast.TypeDefErrorExpr{Payload: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}},
		Op:    ast.TokenBitwiseOr,
		Right: ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: &aName}},
	})
	if err == nil {
		t.Fatal("expected error from failed left subexpression")
	}
}

func TestTypeDefExprToTypeNode_binaryPropagatesRightError(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	aName := ast.TypeIdent("A")
	_, err := tc.TypeDefExprToTypeNode(ast.TypeDefBinaryExpr{
		Left:  ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: &aName}},
		Op:    ast.TokenBitwiseOr,
		Right: ast.TypeDefErrorExpr{Payload: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}},
	})
	if err == nil {
		t.Fatal("expected error from failed right subexpression")
	}
}

func TestTypeDefExprToTypeNode_assertionMissingAssertion(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	_, err := tc.TypeDefExprToTypeNode(ast.TypeDefAssertionExpr{Assertion: nil})
	if err == nil {
		t.Fatal("expected error when assertion is nil")
	}
}

func TestTypeDefExprToTypeNode_assertionWithConstraintsUsesAssertionType(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	str := ast.TypeString
	u, err := tc.TypeDefExprToTypeNode(ast.TypeDefAssertionExpr{
		Assertion: &ast.AssertionNode{
			BaseType: &str,
			Constraints: []ast.ConstraintNode{
				{Name: "Guard"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if u.Ident != ast.TypeAssertion {
		t.Fatalf("expected assertion type node, got %+v", u)
	}
}

func TestTypeDefExprToTypeNode_conjunctiveIdenticalMeetTypes(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	str := ast.TypeString
	got, err := tc.TypeDefExprToTypeNode(ast.TypeDefBinaryExpr{
		Left:  ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: &str}},
		Op:    ast.TokenBitwiseAnd,
		Right: ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: &str}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Ident != ast.TypeString {
		t.Fatalf("String & String should meet to String, got %+v", got)
	}
}

func TestTypeDefExprToTypeNode_conjunctiveNominalErrorAndErrorMeet(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	tc.registerType(ast.TypeDefNode{
		Ident: "NE",
		Expr: ast.TypeDefErrorExpr{
			Payload: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{
				"k": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
			}},
		},
	})
	ne := ast.TypeIdent("NE")
	errT := ast.TypeError
	got, err := tc.TypeDefExprToTypeNode(ast.TypeDefBinaryExpr{
		Left:  ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: &ne}},
		Op:    ast.TokenBitwiseAnd,
		Right: ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: &errT}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Ident != "NE" {
		t.Fatalf("NE & Error should meet to NE via subtyping, got %+v", got)
	}
}

func TestMeetTypesSubtyping_ErrorAndNominal_reverseOrder(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	tc.registerType(ast.TypeDefNode{
		Ident: "NE",
		Expr: ast.TypeDefErrorExpr{
			Payload: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{
				"k": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
			}},
		},
	})
	got, ok := tc.MeetTypesSubtyping(ast.TypeNode{Ident: ast.TypeError}, ast.TypeNode{Ident: "NE"})
	if !ok || got.Ident != "NE" {
		t.Fatalf("MeetTypesSubtyping(Error, NE) want NE, got %+v ok=%v", got, ok)
	}
}

func TestValidateTypeDefBinary_wrapsLowerError(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	aName := ast.TypeIdent("A")
	err := tc.validateTypeDefBinary("MyAlias", ast.TypeDefBinaryExpr{
		Left:  ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: &aName}},
		Op:    ast.TokenBitwiseOr,
		Right: ast.TypeDefErrorExpr{Payload: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}},
	})
	if err == nil || !strings.Contains(err.Error(), "MyAlias") {
		t.Fatalf("expected wrapped error mentioning typedef name, got %v", err)
	}
}

func TestExpandTypeDefBinaryIfNeeded_cases(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)

	if _, ok := tc.expandTypeDefBinaryIfNeeded(ast.TypeNode{Ident: ""}); ok {
		t.Fatal("empty ident should not expand")
	}
	if _, ok := tc.expandTypeDefBinaryIfNeeded(ast.TypeNode{Ident: "NoSuch"}); ok {
		t.Fatal("missing typedef should not expand")
	}

	tc.registerType(ast.TypeDefNode{
		Ident: "Plain",
		Expr:  ast.TypeDefShapeExpr{Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}},
	})
	if _, ok := tc.expandTypeDefBinaryIfNeeded(ast.TypeNode{Ident: "Plain"}); ok {
		t.Fatal("non-binary typedef body should not expand")
	}

	tc.registerType(ast.TypeDefNode{
		Ident: "BadBin",
		Expr: ast.TypeDefBinaryExpr{
			Left:  ast.TypeDefErrorExpr{Payload: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}},
			Op:    ast.TokenBitwiseOr,
			Right: ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: ptrIdent("A")}},
		},
	})
	if _, ok := tc.expandTypeDefBinaryIfNeeded(ast.TypeNode{Ident: "BadBin"}); ok {
		t.Fatal("typedef whose body does not lower should not expand")
	}

	tc.registerType(ast.TypeDefNode{Ident: "A", Expr: ast.TypeDefErrorExpr{Payload: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}}})
	tc.registerType(ast.TypeDefNode{
		Ident: "SelfU",
		Expr: ast.TypeDefBinaryExpr{
			Left:  ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: ptrIdent("SelfU")}},
			Op:    ast.TokenBitwiseOr,
			Right: ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: ptrIdent("A")}},
		},
	})
	if _, ok := tc.expandTypeDefBinaryIfNeeded(ast.TypeNode{Ident: "SelfU"}); ok {
		t.Fatal("self-referential binary typedef should not expand (avoids infinite expand)")
	}

	tc.registerType(ast.TypeDefNode{
		Ident: "AB",
		Expr: ast.TypeDefBinaryExpr{
			Left:  ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: ptrIdent("A")}},
			Op:    ast.TokenBitwiseOr,
			Right: ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: ptrIdent("B")}},
		},
	})
	tc.registerType(ast.TypeDefNode{Ident: "B", Expr: ast.TypeDefErrorExpr{Payload: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}}})
	canon, ok := tc.expandTypeDefBinaryIfNeeded(ast.TypeNode{Ident: "AB"})
	if !ok || canon.Ident != ast.TypeUnion || len(canon.TypeParams) != 2 {
		t.Fatalf("expected expanded union, got ok=%v %+v", ok, canon)
	}
}

func TestIsErrorKindedType_namedTypedefExpandsToErrorUnion(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	tc.registerType(ast.TypeDefNode{Ident: "A", Expr: ast.TypeDefErrorExpr{Payload: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}}})
	tc.registerType(ast.TypeDefNode{Ident: "B", Expr: ast.TypeDefErrorExpr{Payload: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}}})
	tc.registerType(ast.TypeDefNode{
		Ident: "AB",
		Expr: ast.TypeDefBinaryExpr{
			Left:  ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: ptrIdent("A")}},
			Op:    ast.TokenBitwiseOr,
			Right: ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: ptrIdent("B")}},
		},
	})
	if !tc.IsErrorKindedType(ast.TypeNode{Ident: "AB"}) {
		t.Fatal("IsErrorKindedType should expand named typedef AB to a union of nominal errors")
	}
}
