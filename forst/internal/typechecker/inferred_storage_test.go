package typechecker

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

// Tests for inferred_storage.go (storeInferredType, storeInferredFunctionReturnType).

// Regression: hasEnsure must count *ast.EnsureNode (interface assertion to ast.EnsureNode alone
// misses pointers).
func TestInferFunctionReturnType_pointerEnsureNode_skipsLegacyErrorAppend(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	tc := New(log, false)

	intT := ast.TypeNode{Ident: ast.TypeInt}
	errT := ast.TypeNode{Ident: ast.TypeError}
	res := ast.TypeNode{Ident: ast.TypeResult, TypeParams: []ast.TypeNode{intT, errT}}

	fn := ast.FunctionNode{
		Ident:       ast.Ident{ID: "okInt"},
		ReturnTypes: []ast.TypeNode{res},
		Body: []ast.Node{
			ast.AssignmentNode{
				IsShort: true,
				LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "n"}}},
				RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: 42}},
			},
			&ast.EnsureNode{
				Variable: ast.VariableNode{Ident: ast.Ident{ID: "n"}},
				Assertion: ast.AssertionNode{
					BaseType: typeIdentPtr(string(ast.TypeInt)),
				},
			},
			ast.ReturnNode{Values: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "n"}}}},
		},
	}

	tc.pushScope(&fn)
	defer tc.popScope()
	tc.CurrentScope().RegisterSymbol(ast.Identifier("n"), []ast.TypeNode{{Ident: ast.TypeInt}}, SymbolVariable)

	tc.Functions[fn.Ident.ID] = FunctionSignature{
		Ident:       fn.Ident,
		ReturnTypes: []ast.TypeNode{res},
	}

	got, err := tc.inferFunctionReturnType(fn)
	if err != nil {
		t.Fatal(err)
	}
	// Constructor-free success: inference yields plain Int; Result(...) is the parsed/signature type.
	if len(got) != 1 || got[0].Ident != ast.TypeInt {
		t.Fatalf("got %v, want Int", formatTypeList(got))
	}

	tc.storeInferredFunctionReturnType(&fn, got)
	stored := tc.Functions[fn.Ident.ID]
	if len(stored.ReturnTypes) != 1 || !stored.ReturnTypes[0].IsResultType() {
		t.Fatalf("stored return types %v, want single Result(...)", formatTypeList(stored.ReturnTypes))
	}
}
