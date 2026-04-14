package transformergo

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func TestTransformEnsureCondition_unknownTypeGuardWithoutConstraints_returnsNoStatements(t *testing.T) {
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tc.CurrentScope().RegisterSymbol("x", []ast.TypeNode{{Ident: ast.TypeString}}, typechecker.SymbolVariable)
	tr := setupTransformer(tc, log)

	unknown := ast.TypeIdent("NoSuchGuard")
	ensure := &ast.EnsureNode{
		Variable: ast.VariableNode{Ident: ast.Ident{ID: "x"}},
		Assertion: ast.AssertionNode{
			BaseType:    &unknown,
			Constraints: nil,
		},
	}

	stmts, err := tr.transformEnsureCondition(ensure)
	if err != nil {
		t.Fatalf("transformEnsureCondition: %v", err)
	}
	if len(stmts) != 0 {
		t.Fatalf("expected no statements for unresolved bare type guard, got %d", len(stmts))
	}
}

func TestTransformEnsureCondition_constraintProducesExprStmt(t *testing.T) {
	log := logrus.New()
	tc := setupTypeChecker(log)
	tc.CurrentScope().RegisterSymbol("ok", []ast.TypeNode{{Ident: ast.TypeBool}}, typechecker.SymbolVariable)
	tr := setupTransformer(tc, log)

	ensure := &ast.EnsureNode{
		Variable: ast.VariableNode{Ident: ast.Ident{ID: "ok"}},
		Assertion: ast.AssertionNode{
			Constraints: []ast.ConstraintNode{{Name: string(TrueConstraint)}},
		},
	}

	stmts, err := tr.transformEnsureCondition(ensure)
	if err != nil {
		t.Fatalf("transformEnsureCondition: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected one transformed statement, got %d", len(stmts))
	}
}
