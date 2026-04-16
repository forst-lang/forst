package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestBuiltinTypeHelperPredicates(t *testing.T) {
	t.Parallel()
	if !clearOperandAllowed(ast.TypeNode{Ident: ast.TypeMap}) {
		t.Fatal("expected map to be clear()-allowed")
	}
	if clearOperandAllowed(ast.TypeNode{Ident: ast.TypeString}) {
		t.Fatal("string must not be clear()-allowed")
	}
	if !isScalarTypeIdent(ast.TypeInt) || isScalarTypeIdent(ast.TypeMap) {
		t.Fatal("isScalarTypeIdent mismatch")
	}
}

func TestDispatchCloseClearPanic_basicPaths(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)

	if _, _, err := tc.dispatchClose(nil, nil, ast.SourceSpan{}); err == nil {
		t.Fatal("expected dispatchClose arity error")
	}
	if _, _, err := tc.dispatchPanic(nil, nil, ast.SourceSpan{}); err == nil {
		t.Fatal("expected dispatchPanic arity error")
	}
	if _, _, err := tc.dispatchClear([]ast.ExpressionNode{ast.StringLiteralNode{Value: "x"}}, nil, ast.SourceSpan{}); err == nil {
		t.Fatal("expected dispatchClear type error")
	}
}

func TestTypeRegistryStructuralLookups(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	shape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	tc.Defs["User"] = ast.TypeDefNode{Ident: "User", Expr: ast.TypeDefShapeExpr{Shape: shape}}
	tc.Defs["T_hash"] = ast.TypeDefNode{Ident: "T_hash", Expr: ast.TypeDefShapeExpr{Shape: shape}}

	if got := tc.FindStructurallyIdenticalNamedType(ast.TypeNode{Ident: "T_hash", TypeKind: ast.TypeKindHashBased}); got != "User" {
		t.Fatalf("FindStructurallyIdenticalNamedType: got %q", got)
	}
	if got := tc.FindAnyStructurallyIdenticalNamedType(shape); got == "" {
		t.Fatal("FindAnyStructurallyIdenticalNamedType should find User")
	}
}

func TestRegisterTypeIfMissing_andTypeGuardConstraint(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	tc.RegisterTypeIfMissing("X", ast.TypeDefNode{Ident: "X", Expr: ast.TypeDefShapeExpr{Shape: ast.ShapeNode{}}})
	tc.RegisterTypeIfMissing("X", ast.TypeDefNode{Ident: "X", Expr: ast.TypeDefShapeExpr{Shape: ast.ShapeNode{}}})
	if _, ok := tc.Defs["X"]; !ok {
		t.Fatal("expected X registered")
	}
	guard := ast.TypeGuardNode{Ident: "Strong"}
	tc.Defs["Strong"] = guard
	if !tc.IsTypeGuardConstraint("Strong") {
		t.Fatal("expected Strong as type-guard constraint")
	}
	if tc.IsTypeGuardConstraint("Missing") {
		t.Fatal("missing guard should not be a type-guard constraint")
	}
}

func TestEnsureMatching_mismatchMessage(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}}
	_, err := ensureMatching(tc, fn, []ast.TypeNode{{Ident: ast.TypeInt}}, []ast.TypeNode{{Ident: ast.TypeString}}, "test")
	if err == nil {
		t.Fatal("expected mismatch")
	}
	if !strings.Contains(err.Error(), "Type mismatch") {
		t.Fatalf("unexpected ensureMatching error: %v", err)
	}
}

