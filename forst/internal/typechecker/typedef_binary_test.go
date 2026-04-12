package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestTypeDefExprToTypeNode_nilExprErrors(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	_, err := tc.TypeDefExprToTypeNode(nil)
	if err == nil {
		t.Fatal("expected error for nil typedef expression")
	}
}

func TestTypeDefExprToTypeNode_unionAndMeet(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	e1name := ast.TypeIdent("E1")
	e2name := ast.TypeIdent("E2")
	tc.registerType(ast.TypeDefNode{
		Ident: "E1",
		Expr: ast.TypeDefErrorExpr{
			Payload: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{
				"a": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
			}},
		},
	})
	tc.registerType(ast.TypeDefNode{
		Ident: "E2",
		Expr: ast.TypeDefErrorExpr{
			Payload: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{
				"b": {Type: &ast.TypeNode{Ident: ast.TypeString}},
			}},
		},
	})

	unionExpr := ast.TypeDefBinaryExpr{
		Left: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{BaseType: &e1name},
		},
		Op: ast.TokenBitwiseOr,
		Right: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{BaseType: &e2name},
		},
	}
	u, err := tc.TypeDefExprToTypeNode(unionExpr)
	if err != nil {
		t.Fatal(err)
	}
	if u.Ident != ast.TypeUnion || len(u.TypeParams) != 2 {
		t.Fatalf("want union of E1 E2, got %+v", u)
	}
	if !tc.IsErrorKindedType(u) {
		t.Fatal("union of nominal errors should be error-kinded")
	}
	if !tc.IsTypeCompatible(u, ast.TypeNode{Ident: ast.TypeError}) {
		t.Fatal("union of error nominals should be assignable to Error")
	}

	str := ast.TypeString
	intT := ast.TypeInt
	_, err = tc.TypeDefExprToTypeNode(ast.TypeDefBinaryExpr{
		Left:  ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: &str}},
		Op:    ast.TokenBitwiseAnd,
		Right: ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: &intT}},
	})
	if err == nil {
		t.Fatal("expected error for String & Int")
	}
}

func TestMeetTypesSubtyping_errorWidth(t *testing.T) {
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
	got, ok := tc.MeetTypesSubtyping(ast.TypeNode{Ident: "NE"}, ast.TypeNode{Ident: ast.TypeError})
	if !ok || got.Ident != "NE" {
		t.Fatalf("MeetTypesSubtyping: got %+v ok=%v", got, ok)
	}
}

func TestJoinTypes_flattensNestedUnion(t *testing.T) {
	t.Parallel()
	u := ast.NewUnionType(
		ast.TypeNode{Ident: ast.TypeString},
		ast.TypeNode{Ident: ast.TypeInt},
	)
	got, ok := JoinTypes([]ast.TypeNode{u, ast.TypeNode{Ident: ast.TypeBool}})
	if !ok || got.Ident != ast.TypeUnion || len(got.TypeParams) != 3 {
		t.Fatalf("got %+v ok=%v", got, ok)
	}
}

func TestNewUnionType_singleMemberCollapses(t *testing.T) {
	t.Parallel()
	u := ast.NewUnionType(ast.TypeNode{Ident: ast.TypeString})
	if u.Ident != ast.TypeString {
		t.Fatalf("expected collapse to String, got %+v", u)
	}
}

func TestIsTypeCompatible_unionRightSumIntro(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	tc.registerType(ast.TypeDefNode{
		Ident: "E1",
		Expr: ast.TypeDefErrorExpr{
			Payload: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{
				"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
			}},
		},
	})
	u := ast.NewUnionType(ast.TypeNode{Ident: "E1"}, ast.TypeNode{Ident: ast.TypeString})
	if !tc.IsTypeCompatible(ast.TypeNode{Ident: "E1"}, u) {
		t.Fatal("E1 <: E1 | String")
	}
	if !tc.IsTypeCompatible(ast.TypeNode{Ident: ast.TypeString}, u) {
		t.Fatal("String <: E1 | String (second arm)")
	}
	if tc.IsTypeCompatible(ast.TypeNode{Ident: ast.TypeInt}, u) {
		t.Fatal("Int should not be assignable to E1 | String")
	}
}

func TestCheckTypes_unionErrorTypedef_source(t *testing.T) {
	t.Parallel()
	log := ast.SetupTestLogger(nil)
	src := `package main

error A { x: Int }
error B { y: String }

type AB = A | B

func main() {}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
}

func TestExpandTypeDefBinaryIfNeeded_roundTrip(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	tc.registerType(ast.TypeDefNode{
		Ident: "AB",
		Expr: ast.TypeDefBinaryExpr{
			Left:  ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: ptrIdent("A")}},
			Op:    ast.TokenBitwiseOr,
			Right: ast.TypeDefAssertionExpr{Assertion: &ast.AssertionNode{BaseType: ptrIdent("B")}},
		},
	})
	tc.registerType(ast.TypeDefNode{
		Ident: "A",
		Expr: ast.TypeDefErrorExpr{Payload: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{
			"k": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
		}}},
	})
	tc.registerType(ast.TypeDefNode{
		Ident: "B",
		Expr: ast.TypeDefErrorExpr{Payload: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{
			"k": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		}}},
	})
	if !tc.IsTypeCompatible(ast.TypeNode{Ident: "AB"}, ast.TypeNode{Ident: ast.TypeError}) {
		t.Fatal("named typedef binary should expand to union and assign to Error")
	}
}

func TestIsErrorKindedType_unionAndIntersectionBranches(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	if tc.IsErrorKindedType(ast.TypeNode{Ident: ast.TypeUnion, TypeParams: nil}) {
		t.Fatal("empty union should not be error-kinded")
	}
	if tc.IsErrorKindedType(ast.TypeNode{Ident: ast.TypeUnion, TypeParams: []ast.TypeNode{}}) {
		t.Fatal("union with zero members should not be error-kinded")
	}
	// Mixed non-error member => false
	if tc.IsErrorKindedType(ast.NewUnionType(ast.TypeNode{Ident: ast.TypeError}, ast.TypeNode{Ident: ast.TypeString})) {
		t.Fatal("Error | String is not fully error-kinded (String is not)")
	}
	tc.registerType(ast.TypeDefNode{
		Ident: "E1",
		Expr: ast.TypeDefErrorExpr{
			Payload: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{
				"a": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
			}},
		},
	})
	tc.registerType(ast.TypeDefNode{
		Ident: "E2",
		Expr: ast.TypeDefErrorExpr{
			Payload: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{
				"b": {Type: &ast.TypeNode{Ident: ast.TypeString}},
			}},
		},
	})
	if !tc.IsErrorKindedType(ast.NewUnionType(ast.TypeNode{Ident: "E1"}, ast.TypeNode{Ident: "E2"})) {
		t.Fatal("E1 | E2 nominal errors should be error-kinded")
	}
	if !tc.IsErrorKindedType(ast.NewIntersectionType(ast.TypeNode{Ident: "E1"}, ast.TypeNode{Ident: "E2"})) {
		t.Fatal("E1 & E2 (both nominal errors) should be error-kinded for IsErrorKindedType")
	}
}

func ptrIdent(id string) *ast.TypeIdent {
	t := ast.TypeIdent(id)
	return &t
}
