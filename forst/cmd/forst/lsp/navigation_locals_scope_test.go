package lsp

import (
	"testing"

	"forst/internal/ast"
)

func TestNavigationLocals_lookupSymbolAndSameBinding(t *testing.T) {
	t.Parallel()
	const src = `package main

func outer(x Int): Int {
	y := x
	return y
}

func main() {
	println(outer(1))
}
`
	nodes, tokens, tc := parseNodesAndTokensForNavigationTest(t, src)
	fn := firstFunctionNode(t, nodes)
	body := fn.Body
	// Find second `x` token (use in y := x).
	posUse, ok := lspPosForNthIdentToken(tokens, "x", 1)
	if !ok {
		t.Fatal("use of x not found")
	}
	idx := tokenIndexAtLSPPosition(tokens, posUse)
	if idx < 0 {
		t.Fatal("token index")
	}
	sym, ok := lookupSymbolAtToken(tc, nodes, tokens, idx, ast.Identifier("x"))
	if !ok {
		t.Fatal("expected symbol for x")
	}
	paramTok := findParamIdentTokenForFunction(tokens, fn, "x")
	if paramTok == nil {
		t.Fatal("param token")
	}
	paramIdx := tokenIndexAtLSPPosition(tokens, LSPPosition{Line: paramTok.Line - 1, Character: paramTok.Column - 1})
	paramSym, ok := lookupSymbolAtToken(tc, nodes, tokens, paramIdx, ast.Identifier("x"))
	if !ok {
		t.Fatal("param symbol")
	}
	if !sameBinding(sym, paramSym) {
		t.Fatalf("use and param should be same binding: %#v vs %#v", sym, paramSym)
	}
	_ = body
}

func TestNavigationLocals_definingTokenFromFunctionParam(t *testing.T) {
	t.Parallel()
	const src = `package main

func f(name String): String {
	return name
}
`
	nodes, tokens, tc := parseNodesAndTokensForNavigationTest(t, src)
	fn := firstFunctionNode(t, nodes)
	usePos, ok := lspPosForNthIdentToken(tokens, "name", 1)
	if !ok {
		t.Fatal("use of name")
	}
	idx := tokenIndexAtLSPPosition(tokens, usePos)
	tok := &tokens[idx]
	def := definingTokenForLocalBinding(&forstDocumentContext{
		Nodes: nodes,
		Tokens: tokens,
		TC:     tc,
	}, idx, tok)
	if def == nil {
		t.Fatal("expected defining token for parameter")
	}
	if def.Value != "name" {
		t.Fatalf("def = %q", def.Value)
	}
	_ = fn
}
