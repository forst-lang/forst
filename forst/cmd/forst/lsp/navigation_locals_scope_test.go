package lsp

import (
	"testing"

	"forst/internal/ast"
)

func TestNavigationLocals_lookupSymbolAtToken_afterCheckTypes(t *testing.T) {
	t.Parallel()
	const src = `package main

func outer(x Int): Int {
	y := x
	return y
}
`
	nodes, tokens, tc := parseNodesAndTokensForNavigationTest(t, src)
	usePos, ok := lspPosForNthIdentToken(tokens, "x", 1)
	if !ok {
		t.Fatal("use of x")
	}
	idx := tokenIndexAtLSPPosition(tokens, usePos)
	if err := tc.RestoreScope(nil); err != nil {
		t.Fatalf("RestoreScope(nil): %v", err)
	}
	sym, ok := lookupSymbolAtToken(tc, nodes, tokens, idx, ast.Identifier("x"))
	if !ok {
		t.Fatal("lookupSymbolAtToken failed")
	}
	if sym.Identifier != "x" {
		t.Fatalf("sym=%#v", sym)
	}
}

func TestNavigationLocals_findInnermostScopeRestores_twoFunctions(t *testing.T) {
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
	usePos, ok := lspPosForNthIdentToken(tokens, "x", 1)
	if !ok {
		t.Fatal("use of x")
	}
	idx := tokenIndexAtLSPPosition(tokens, usePos)
	scopeNode := findInnermostScopeNode(nodes, tokens, idx, tc)
	if scopeNode == nil {
		t.Fatal("expected scope node")
	}
	if err := tc.RestoreScope(scopeNode); err != nil {
		t.Fatalf("RestoreScope(%T): %v", scopeNode, err)
	}
	if _, ok := tc.CurrentScope().LookupVariable(ast.Identifier("x")); !ok {
		t.Fatal("expected param x after restore")
	}
}

func TestNavigationLocals_findInnermostScopeRestores(t *testing.T) {
	t.Parallel()
	const src = `package main

func f(x Int): Int {
	return x
}
`
	nodes, tokens, tc := parseNodesAndTokensForNavigationTest(t, src)
	usePos, ok := lspPosForNthIdentToken(tokens, "x", 1)
	if !ok {
		t.Fatal("use of x")
	}
	idx := tokenIndexAtLSPPosition(tokens, usePos)
	scopeNode := findInnermostScopeNode(nodes, tokens, idx, tc)
	if scopeNode == nil {
		t.Fatal("expected scope node")
	}
	if err := tc.RestoreScope(scopeNode); err != nil {
		t.Fatalf("RestoreScope(%T): %v", scopeNode, err)
	}
	if _, ok := tc.CurrentScope().LookupVariable(ast.Identifier("x")); !ok {
		t.Fatal("expected param x after restore")
	}
}

func TestNavigationLocals_restoreScopeOnParsedFunctionNode(t *testing.T) {
	t.Parallel()
	const src = `package main

func f(x Int): Int {
	return x
}
`
	nodes, _, tc := parseNodesAndTokensForNavigationTest(t, src)
	for _, n := range nodes {
		fn, ok := n.(ast.FunctionNode)
		if !ok || fn.Ident.ID != "f" {
			continue
		}
		if err := tc.RestoreScope(n); err != nil {
			t.Fatalf("RestoreScope: %v", err)
		}
		if _, ok := tc.CurrentScope().LookupVariable(ast.Identifier("x")); !ok {
			t.Fatal("expected param x in function scope")
		}
		return
	}
	t.Fatal("function f not found")
}

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
