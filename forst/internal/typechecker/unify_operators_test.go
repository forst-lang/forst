package typechecker

import (
	"forst/internal/ast"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestUnifyTypes_unaryIncrementDoesNotInferNilOperand(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.scopeStack.globalScope().RegisterSymbol("i", []ast.TypeNode{{Ident: ast.TypeInt}}, SymbolVariable)
	tc.VariableTypes["i"] = []ast.TypeNode{{Ident: ast.TypeInt}}

	operand := ast.VariableNode{Ident: ast.Ident{ID: "i"}}
	ty, err := tc.unifyTypes(operand, nil, ast.TokenPlusPlus)
	if err != nil {
		t.Fatal(err)
	}
	if ty.Ident != ast.TypeInt {
		t.Fatalf("got %s", ty.Ident)
	}
}

func TestUnifyTypes_unaryDecrement(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.scopeStack.globalScope().RegisterSymbol("x", []ast.TypeNode{{Ident: ast.TypeFloat}}, SymbolVariable)
	tc.VariableTypes["x"] = []ast.TypeNode{{Ident: ast.TypeFloat}}
	ty, err := tc.unifyTypes(ast.VariableNode{Ident: ast.Ident{ID: "x"}}, nil, ast.TokenMinusMinus)
	if err != nil {
		t.Fatal(err)
	}
	if ty.Ident != ast.TypeFloat {
		t.Fatalf("got %s", ty.Ident)
	}
}

func TestUnifyTypes_unaryMinusInt(t *testing.T) {
	tc := New(logrus.New(), false)
	ty, err := tc.unifyTypes(ast.IntLiteralNode{Value: 1}, nil, ast.TokenMinus)
	if err != nil {
		t.Fatal(err)
	}
	if ty.Ident != ast.TypeInt {
		t.Fatalf("got %s", ty.Ident)
	}
}

func TestUnifyTypes_unaryLogicalNot(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.scopeStack.globalScope().RegisterSymbol("b", []ast.TypeNode{{Ident: ast.TypeBool}}, SymbolVariable)
	tc.VariableTypes["b"] = []ast.TypeNode{{Ident: ast.TypeBool}}
	ty, err := tc.unifyTypes(ast.VariableNode{Ident: ast.Ident{ID: "b"}}, nil, ast.TokenLogicalNot)
	if err != nil {
		t.Fatal(err)
	}
	if ty.Ident != ast.TypeBool {
		t.Fatalf("got %s", ty.Ident)
	}
}

func TestUnifyTypes_unaryLogicalNotRejectsNonBool(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.scopeStack.globalScope().RegisterSymbol("n", []ast.TypeNode{{Ident: ast.TypeInt}}, SymbolVariable)
	tc.VariableTypes["n"] = []ast.TypeNode{{Ident: ast.TypeInt}}
	_, err := tc.unifyTypes(ast.VariableNode{Ident: ast.Ident{ID: "n"}}, nil, ast.TokenLogicalNot)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUnifyTypes_unaryIncrementRejectsString(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.scopeStack.globalScope().RegisterSymbol("s", []ast.TypeNode{{Ident: ast.TypeString}}, SymbolVariable)
	tc.VariableTypes["s"] = []ast.TypeNode{{Ident: ast.TypeString}}
	_, err := tc.unifyTypes(ast.VariableNode{Ident: ast.Ident{ID: "s"}}, nil, ast.TokenPlusPlus)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUnifyTypes_unsupportedUnaryWithNilRight(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.scopeStack.globalScope().RegisterSymbol("i", []ast.TypeNode{{Ident: ast.TypeInt}}, SymbolVariable)
	tc.VariableTypes["i"] = []ast.TypeNode{{Ident: ast.TypeInt}}
	_, err := tc.unifyTypes(ast.VariableNode{Ident: ast.Ident{ID: "i"}}, nil, ast.TokenStar)
	if err == nil {
		t.Fatal("expected error for unsupported unary")
	}
}

func TestUnifyTypes_comparisonEqNil(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.scopeStack.globalScope().RegisterSymbol("err", []ast.TypeNode{{Ident: ast.TypeError}}, SymbolVariable)
	tc.VariableTypes["err"] = []ast.TypeNode{{Ident: ast.TypeError}}
	ty, err := tc.unifyTypes(ast.VariableNode{Ident: ast.Ident{ID: "err"}}, ast.NilLiteralNode{}, ast.TokenEquals)
	if err != nil {
		t.Fatal(err)
	}
	if ty.Ident != ast.TypeBool {
		t.Fatalf("want Bool, got %s", ty.Ident)
	}
}

func TestUnifyTypes_comparisonNilEqErr(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.scopeStack.globalScope().RegisterSymbol("err", []ast.TypeNode{{Ident: ast.TypeError}}, SymbolVariable)
	tc.VariableTypes["err"] = []ast.TypeNode{{Ident: ast.TypeError}}
	ty, err := tc.unifyTypes(ast.NilLiteralNode{}, ast.VariableNode{Ident: ast.Ident{ID: "err"}}, ast.TokenEquals)
	if err != nil {
		t.Fatal(err)
	}
	if ty.Ident != ast.TypeBool {
		t.Fatalf("want Bool, got %s", ty.Ident)
	}
}

func TestUnifyTypes_comparisonNilEqNil(t *testing.T) {
	tc := New(logrus.New(), false)
	ty, err := tc.unifyTypes(ast.NilLiteralNode{}, ast.NilLiteralNode{}, ast.TokenEquals)
	if err != nil {
		t.Fatal(err)
	}
	if ty.Ident != ast.TypeBool {
		t.Fatalf("want Bool, got %s", ty.Ident)
	}
}

func TestUnifyTypes_comparisonErrNeNil(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.scopeStack.globalScope().RegisterSymbol("err", []ast.TypeNode{{Ident: ast.TypeError}}, SymbolVariable)
	tc.VariableTypes["err"] = []ast.TypeNode{{Ident: ast.TypeError}}
	ty, err := tc.unifyTypes(ast.VariableNode{Ident: ast.Ident{ID: "err"}}, ast.NilLiteralNode{}, ast.TokenNotEquals)
	if err != nil {
		t.Fatal(err)
	}
	if ty.Ident != ast.TypeBool {
		t.Fatalf("want Bool, got %s", ty.Ident)
	}
}
