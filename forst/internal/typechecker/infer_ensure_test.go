package typechecker

import (
	"io"
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

// TestInferEnsureType_validatesConstraintsLikeBinaryIs ensures ensure assertions
// use the same constraint validation as binary `is` (validateAssertionNode).
func TestInferEnsureType_validatesConstraintsLikeBinaryIs(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	t.Run("type_guard_subject_mismatch", func(t *testing.T) {
		tc := New(log, false)
		registerTypeGuardExpectsInt(tc, ast.TypeIdent("ExpectsInt"))
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
		registerTypeGuardExpectsInt(tc, ast.TypeIdent("ExpectsInt"))
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

	t.Run("Valid_reserved_placeholder", func(t *testing.T) {
		tc := New(log, false)
		fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
		tc.scopeStack.pushScope(fn)
		tc.CurrentScope().RegisterSymbol(ast.Identifier("s"), []ast.TypeNode{{Ident: ast.TypeString}}, SymbolVariable)

		ensure := ast.EnsureNode{
			Variable: ast.VariableNode{Ident: ast.Ident{ID: "s"}},
			Assertion: ast.AssertionNode{
				Constraints: []ast.ConstraintNode{
					{Name: "Valid", Args: []ast.ConstraintArgumentNode{}},
				},
			},
		}
		_, err := tc.inferEnsureType(ensure)
		if err == nil {
			t.Fatal("expected error: Valid() is reserved")
		}
		if !strings.Contains(err.Error(), "Valid() is a reserved placeholder") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Present_allows_pointer", func(t *testing.T) {
		tc := New(log, false)
		fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
		tc.scopeStack.pushScope(fn)
		ptrToStr := ast.NewPointerType(ast.TypeNode{Ident: ast.TypeString})
		tc.CurrentScope().RegisterSymbol(ast.Identifier("s"), []ast.TypeNode{ptrToStr}, SymbolVariable)

		ensure := ast.EnsureNode{
			Variable: ast.VariableNode{Ident: ast.Ident{ID: "s"}},
			Assertion: ast.AssertionNode{
				Constraints: []ast.ConstraintNode{
					{Name: "Present", Args: []ast.ConstraintArgumentNode{}},
				},
			},
		}
		if _, err := tc.inferEnsureType(ensure); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// TestCheckTypes_ensureBlockNestedControlFlow verifies inferNodeTypes restores ensure-block
// scope once while still typechecking nested if/for statements inside the block.
func TestCheckTypes_ensureBlockNestedControlFlow(t *testing.T) {
	t.Parallel()
	src := `package demo

func F(): Result(String, Error) {
	s := "hello"
	ensure s is Min(1) {
		if s != "" {
			for i := 0; i < 2; i++ {
				s = s + "!"
			}
		}
		return s
	}
	return s
}
`
	log := logrus.New()
	log.SetOutput(io.Discard)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
}
