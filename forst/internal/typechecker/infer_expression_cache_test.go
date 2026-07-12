package typechecker

import (
	"io"
	"path/filepath"
	"testing"

	"forst/internal/ast"
	"forst/internal/forstpkg"

	"github.com/sirupsen/logrus"
)

func TestLookupCachedExpressionTypes_literalAfterInfer(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetOutput(io.Discard)
	tc := New(log, false)
	lit := ast.IntLiteralNode{Value: 42}
	if _, err := tc.inferExpressionType(lit); err != nil {
		t.Fatalf("infer: %v", err)
	}
	types, ok, err := tc.lookupCachedExpressionTypes(lit)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if !ok {
		t.Fatal("expected cache hit after infer")
	}
	if len(types) != 1 || types[0].Ident != ast.TypeInt {
		t.Fatalf("cached type: %#v", types)
	}
}

func TestInferExpressionType_binaryDoesNotUseStructuralCache(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetOutput(io.Discard)
	tc := New(log, false)
	add := ast.BinaryExpressionNode{
		Left:     ast.IntLiteralNode{Value: 1},
		Right:    ast.IntLiteralNode{Value: 2},
		Operator: "+",
	}
	if _, ok, err := tc.lookupCachedExpressionTypes(add); err != nil {
		t.Fatalf("lookup: %v", err)
	} else if ok {
		t.Fatal("binary expression should not hit cache before inference")
	}
}

func TestLookupCachedExpressionTypes_skipsVariableNode(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetOutput(io.Discard)
	tc := New(log, false)
	x := ast.VariableNode{Ident: ast.Ident{ID: "x"}}
	if _, ok, err := tc.lookupCachedExpressionTypes(x); err != nil {
		t.Fatalf("lookupCachedExpressionTypes: %v", err)
	} else if ok {
		t.Fatal("expected no cache hit before any inference")
	}
}

func TestInferExpressionType_variableSkipsCacheDuringInference(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetOutput(io.Discard)
	tc := New(log, false)
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: []ast.Node{}}
	tc.scopeStack.pushScope(fn)
	tc.CurrentScope().RegisterSymbol(ast.Identifier("x"), []ast.TypeNode{{Ident: ast.TypeInt}}, SymbolVariable)
	v := ast.VariableNode{Ident: ast.Ident{ID: "x"}}
	types, err := tc.inferExpressionType(v)
	if err != nil {
		t.Fatalf("infer variable: %v", err)
	}
	if len(types) != 1 || types[0].Ident != ast.TypeInt {
		t.Fatalf("variable type: %#v", types)
	}
}

func TestCheckTypes_tictactoeEngine_withExpressionCache(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "..", "examples", "in", "tictactoe")
	paths := []string{
		filepath.Join(root, "main", "engine.ft"),
		filepath.Join(root, "main", "server.ft"),
	}
	log := logrus.New()
	log.SetOutput(io.Discard)
	merged, _, err := forstpkg.ParseAndMergePackage(log, paths)
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(merged); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	sig, ok := tc.Functions[ast.Identifier("lineWinner")]
	if !ok {
		t.Fatal("missing lineWinner")
	}
	if len(sig.ReturnTypes) != 1 || sig.ReturnTypes[0].Ident != ast.TypeString {
		t.Fatalf("lineWinner return: %#v", sig.ReturnTypes)
	}
	if sig.ReturnTypes[0].IsUserDefined() {
		t.Fatal("lineWinner return String should not be user-defined")
	}
}
