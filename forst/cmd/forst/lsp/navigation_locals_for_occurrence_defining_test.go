package lsp

import (
	"testing"

	"forst/internal/ast"
)

func nthForNodeInFirstFunction(t *testing.T, nodes []ast.Node, n int) *ast.ForNode {
	t.Helper()
	fn := firstFunctionNode(t, nodes)
	seen := 0
	for _, stmt := range fn.Body {
		f, ok := stmt.(*ast.ForNode)
		if !ok {
			continue
		}
		if seen == n {
			return f
		}
		seen++
	}
	t.Fatalf("for loop #%d not found (only %d for loops)", n, seen)
	return nil
}

func TestForNodeOccurrenceIndex_nthAmongSiblingForLoops(t *testing.T) {
	t.Parallel()
	src := `package main

func f() {
	for i := 0; i < 1; i++ {
		println(string(i))
	}
	for j := 0; j < 1; j++ {
		println(string(j))
	}
}
`
	nodes, _, tc := parseNodesAndTokensForNavigationTest(t, src)
	first := nthForNodeInFirstFunction(t, nodes, 0)
	second := nthForNodeInFirstFunction(t, nodes, 1)
	if got := forNodeOccurrenceIndex(nodes, first, tc.Hasher); got != 0 {
		t.Fatalf("first for loop: want 0, got %d", got)
	}
	if got := forNodeOccurrenceIndex(nodes, second, tc.Hasher); got != 1 {
		t.Fatalf("second for loop: want 1, got %d", got)
	}
}

func TestDefiningTokenForFor_parsedRangeKeyValueAndThreeClause(t *testing.T) {
	t.Parallel()
	src := `package main

func f() {
	xs := [1, 2, 3]
	for idx, val := range xs {
		println(string(idx))
		println(string(val))
	}
	for k := 0; k < 2; k++ {
		println(string(k))
	}
}
`
	nodes, tokens, tc := parseNodesAndTokensForNavigationTest(t, src)
	rangeFor := nthForNodeInFirstFunction(t, nodes, 0)
	threeFor := nthForNodeInFirstFunction(t, nodes, 1)

	idxTok := definingTokenForFor(tc, nodes, tokens, rangeFor, "idx")
	if idxTok == nil || idxTok.Value != "idx" {
		t.Fatalf("range key idx: got %+v", idxTok)
	}
	valTok := definingTokenForFor(tc, nodes, tokens, rangeFor, "val")
	if valTok == nil || valTok.Value != "val" {
		t.Fatalf("range val: got %+v", valTok)
	}
	kTok := definingTokenForFor(tc, nodes, tokens, threeFor, "k")
	if kTok == nil || kTok.Value != "k" {
		t.Fatalf("three-clause init k: got %+v", kTok)
	}
}

func TestDefiningTokenForFor_parsedBareForBodyShortDecl(t *testing.T) {
	t.Parallel()
	src := `package main

func f() {
	for {
		x := 1
		println(string(x))
		break
	}
}
`
	nodes, tokens, tc := parseNodesAndTokensForNavigationTest(t, src)
	forn := nthForNodeInFirstFunction(t, nodes, 0)
	if forn.Init != nil || forn.IsRange {
		t.Fatalf("expected bare for { } without init/range, got init=%v range=%v", forn.Init, forn.IsRange)
	}
	tok := definingTokenForFor(tc, nodes, tokens, forn, "x")
	if tok == nil || tok.Value != "x" {
		t.Fatalf("body short decl x: got %+v", tok)
	}
}

func TestForNodeOccurrenceIndex_wrongPointerNotFound(t *testing.T) {
	t.Parallel()
	src := `package main

func f() {
	for i := 0; i < 1; i++ {
		println(string(i))
	}
}
`
	nodes, _, tc := parseNodesAndTokensForNavigationTest(t, src)
	orphan := &ast.ForNode{}
	if got := forNodeOccurrenceIndex(nodes, orphan, tc.Hasher); got != -1 {
		t.Fatalf("orphan for node: want -1, got %d", got)
	}
}
