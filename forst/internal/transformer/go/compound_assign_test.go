package transformergo

import (
	"go/token"
	"testing"

	"forst/internal/ast"
)

func TestCompoundAssignGoToken(t *testing.T) {
	cases := []struct {
		op   ast.TokenIdent
		want token.Token
	}{
		{ast.TokenPlusEq, token.ADD_ASSIGN},
		{ast.TokenMinusEq, token.SUB_ASSIGN},
		{ast.TokenStarEq, token.MUL_ASSIGN},
		{ast.TokenDivideEq, token.QUO_ASSIGN},
		{ast.TokenModuloEq, token.REM_ASSIGN},
		{ast.TokenBitwiseAndEq, token.AND_ASSIGN},
		{ast.TokenBitwiseOrEq, token.OR_ASSIGN},
	}
	for _, tc := range cases {
		got, ok := compoundAssignGoToken(tc.op)
		if !ok || got != tc.want {
			t.Fatalf("%s: got %v ok=%v", tc.op, got, ok)
		}
	}
	if _, ok := compoundAssignGoToken(ast.TokenIdent("??=")); ok {
		t.Fatal("expected false for unknown op")
	}
}

func TestAssignmentGoToken(t *testing.T) {
	short := ast.AssignmentNode{IsShort: true}
	tok, err := assignmentGoToken(short)
	if err != nil || tok != token.DEFINE {
		t.Fatalf("short decl: %v %v", tok, err)
	}
	regular := ast.AssignmentNode{}
	tok, err = assignmentGoToken(regular)
	if err != nil || tok != token.ASSIGN {
		t.Fatalf("regular assign: %v %v", tok, err)
	}
	compound := ast.AssignmentNode{CompoundOp: ast.TokenPlusEq}
	tok, err = assignmentGoToken(compound)
	if err != nil || tok != token.ADD_ASSIGN {
		t.Fatalf("compound assign: %v %v", tok, err)
	}
	_, err = assignmentGoToken(ast.AssignmentNode{CompoundOp: ast.TokenIdent("??=")})
	if err == nil {
		t.Fatal("expected error for unsupported compound op")
	}
}
