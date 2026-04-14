package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestLookupVariableForExpression_prefersFullPathSymbol(t *testing.T) {
	tc := New(logrus.New(), false)
	scope := NewScope(nil, nil, logrus.New())

	scope.RegisterSymbol(ast.Identifier("g"), []ast.TypeNode{{Ident: ast.TypeString}}, SymbolVariable)
	scope.RegisterSymbolWithNarrowing(
		ast.Identifier("g.cells"),
		[]ast.TypeNode{{Ident: ast.TypeInt}},
		SymbolVariable,
		[]string{"GridNarrow"},
		"Min(9).Max(9)",
	)

	varNode := &ast.VariableNode{
		Ident: ast.Ident{ID: ast.Identifier("g.cells")},
	}

	typ, guards, display, err := tc.lookupVariableForExpression(varNode, scope)
	if err != nil {
		t.Fatalf("lookupVariableForExpression error: %v", err)
	}
	if typ.Ident != ast.TypeInt {
		t.Fatalf("expected full-path type %q, got %q", ast.TypeInt, typ.Ident)
	}
	if len(guards) != 1 || guards[0] != "GridNarrow" {
		t.Fatalf("unexpected guards: %#v", guards)
	}
	if display != "Min(9).Max(9)" {
		t.Fatalf("unexpected narrowing predicate display %q", display)
	}
}

func TestLookupVariableForExpression_baseSymbolMustHaveSingleType(t *testing.T) {
	tc := New(logrus.New(), false)
	scope := NewScope(nil, nil, logrus.New())

	scope.RegisterSymbol(
		ast.Identifier("v"),
		[]ast.TypeNode{{Ident: ast.TypeString}, {Ident: ast.TypeInt}},
		SymbolVariable,
	)

	varNode := &ast.VariableNode{
		Ident: ast.Ident{ID: ast.Identifier("v")},
	}

	_, _, _, err := tc.lookupVariableForExpression(varNode, scope)
	if err == nil {
		t.Fatalf("expected error for multi-type variable symbol")
	}
	if !strings.Contains(err.Error(), "expected single type for variable v but got 2 types") {
		t.Fatalf("unexpected error: %v", err)
	}
}
