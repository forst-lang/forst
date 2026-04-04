package typechecker

import (
	"io"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

// TestInferEnsureType_validatesConstraintsLikeBinaryIs ensures ensure assertions
// use the same constraint validation as binary `is` (validateAssertionNode).
func TestInferEnsureType_validatesConstraintsLikeBinaryIs(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	t.Run("type_guard_subject_mismatch", func(t *testing.T) {
		tc := New(log, false)
		tc.Defs[ast.TypeIdent("ExpectsInt")] = ast.TypeGuardNode{
			Ident: "ExpectsInt",
			Subject: ast.SimpleParamNode{
				Ident: ast.Ident{ID: "n"},
				Type:  ast.TypeNode{Ident: ast.TypeInt},
			},
			Body: []ast.Node{},
		}
		fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
		tc.scopeStack.pushScope(fn)
		tc.CurrentScope().RegisterSymbol(ast.Identifier("s"), []ast.TypeNode{{Ident: ast.TypeString}}, SymbolVariable)

		ensure := ast.EnsureNode{
			Variable: ast.VariableNode{Ident: ast.Ident{ID: "s"}},
			Assertion: ast.AssertionNode{
				Constraints: []ast.ConstraintNode{
					{Name: "ExpectsInt", Args: []ast.ConstraintArgumentNode{}},
				},
			},
		}
		_, err := tc.inferEnsureType(ensure)
		if err == nil {
			t.Fatal("expected type mismatch error for ExpectsInt on string variable")
		}
	})

	t.Run("type_guard_subject_ok", func(t *testing.T) {
		tc := New(log, false)
		tc.Defs[ast.TypeIdent("ExpectsInt")] = ast.TypeGuardNode{
			Ident: "ExpectsInt",
			Subject: ast.SimpleParamNode{
				Ident: ast.Ident{ID: "n"},
				Type:  ast.TypeNode{Ident: ast.TypeInt},
			},
			Body: []ast.Node{},
		}
		fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
		tc.scopeStack.pushScope(fn)
		tc.CurrentScope().RegisterSymbol(ast.Identifier("n"), []ast.TypeNode{{Ident: ast.TypeInt}}, SymbolVariable)

		ensure := ast.EnsureNode{
			Variable: ast.VariableNode{Ident: ast.Ident{ID: "n"}},
			Assertion: ast.AssertionNode{
				Constraints: []ast.ConstraintNode{
					{Name: "ExpectsInt", Args: []ast.ConstraintArgumentNode{}},
				},
			},
		}
		if _, err := tc.inferEnsureType(ensure); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Present_requires_pointer", func(t *testing.T) {
		tc := New(log, false)
		fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
		tc.scopeStack.pushScope(fn)
		tc.CurrentScope().RegisterSymbol(ast.Identifier("s"), []ast.TypeNode{{Ident: ast.TypeString}}, SymbolVariable)

		ensure := ast.EnsureNode{
			Variable: ast.VariableNode{Ident: ast.Ident{ID: "s"}},
			Assertion: ast.AssertionNode{
				Constraints: []ast.ConstraintNode{
					{Name: "Present", Args: []ast.ConstraintArgumentNode{}},
				},
			},
		}
		_, err := tc.inferEnsureType(ensure)
		if err == nil {
			t.Fatal("expected error: Present on non-pointer")
		}
	})
}
