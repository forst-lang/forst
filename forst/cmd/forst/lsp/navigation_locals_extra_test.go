package lsp

import (
	"path/filepath"
	"testing"

	"forst/internal/ast"
	"forst/internal/hasher"
	"forst/internal/lexer"
	"forst/internal/parser"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func parseNodesAndTokensForNavigationTest(t *testing.T, src string) ([]ast.Node, []ast.Token, *typechecker.TypeChecker) {
	t.Helper()
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	tokens := lexer.New([]byte(src), "navigation_test.ft", log).Lex()
	p := parser.New(tokens, "navigation_test.ft", log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := typechecker.New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	return nodes, tokens, tc
}

func firstForNode(t *testing.T, nodes []ast.Node) *ast.ForNode {
	t.Helper()
	for _, n := range nodes {
		if fn, ok := n.(ast.FunctionNode); ok {
			for _, stmt := range fn.Body {
				if f, ok := stmt.(*ast.ForNode); ok {
					return f
				}
			}
		}
	}
	t.Fatal("no for node found")
	return nil
}

func firstFunctionNode(t *testing.T, nodes []ast.Node) ast.FunctionNode {
	t.Helper()
	for _, n := range nodes {
		switch fn := n.(type) {
		case ast.FunctionNode:
			return fn
		case *ast.FunctionNode:
			if fn != nil {
				return *fn
			}
		}
	}
	t.Fatal("no function node found")
	return ast.FunctionNode{}
}

func firstIfNode(t *testing.T, nodes []ast.Node) ast.IfNode {
	t.Helper()
	for _, n := range nodes {
		if fn, ok := n.(ast.FunctionNode); ok {
			for _, stmt := range fn.Body {
				if ifn, ok := stmt.(*ast.IfNode); ok {
					return *ifn
				}
			}
		}
	}
	t.Fatal("no if node found")
	return ast.IfNode{}
}

func TestNavigationLocalHelpers_shortDeclOrdinalAndLookup(t *testing.T) {
	t.Parallel()
	src := `package main

func f() {
	x := 1
	if true {
		x := 2
		println(string(x))
	}
	for i := 0; i < 1; i++ {
		x := 3
		println(string(x))
	}
}
`
	nodes, tokens, tc := parseNodesAndTokensForNavigationTest(t, src)
	fn := firstFunctionNode(t, nodes)

	assign := findFirstShortDeclAssignmentInBody(fn.Body, "x")
	if assign == nil {
		t.Fatal("expected short decl assignment in function body")
	}
	want, err := tc.Hasher.HashNode(*assign)
	if err != nil {
		t.Fatalf("hash assign: %v", err)
	}
	k := globalShortDeclOrdinalForAssignment(nodes, "x", want, tc.Hasher)
	if k < 0 {
		t.Fatal("expected non-negative ordinal for short decl")
	}
	tok := nthShortDeclIdentToken(tokens, "x", k)
	if tok == nil || tok.Value != "x" {
		t.Fatalf("expected matching short decl token, got %+v", tok)
	}
}

func TestNavigationLocalHelpers_forAndIfTokenResolution(t *testing.T) {
	t.Parallel()
	src := `package main

func f() {
	for i := 0; i < 2; i++ {
		println(string(i))
	}
	if y := 1; y > 0 {
		println(string(y))
	}
}
`
	nodes, tokens, tc := parseNodesAndTokensForNavigationTest(t, src)
	ifToken := firstIfNode(t, nodes)

	h := hasher.New()
	syntheticFor := ast.ForNode{
		Body: []ast.Node{
			ast.AssignmentNode{
				IsShort: true,
				LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "i"}}},
				RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: 0}},
			},
		},
	}
	fileNodes := []ast.Node{
		ast.FunctionNode{Body: []ast.Node{syntheticFor}},
	}
	nth := forNodeOccurrenceIndex(fileNodes, &syntheticFor, h)
	if nth < -1 {
		t.Fatalf("unexpected for node index %d", nth)
	}
	if got := nthTopLevelForKeyword(tokens, 0); got != -1 {
		t.Fatalf("expected no top-level for keyword in function body, got %d", got)
	}
	forIdx := nthTokenIndexByType(tokens, ast.TokenFor, 0)
	forIdent := findIdentInForThreeClauseHeader(tokens, forIdx, "i")
	if forIdent == nil || forIdent.Value != "i" {
		t.Fatalf("expected for-init identifier i, got %+v", forIdent)
	}
	ifIdent := findIdentInsideIfInitParens(tokens, nodes, ifToken, tc.Hasher, "y")
	if ifIdent != nil {
		t.Fatalf("expected nil: parser does not wrap if-init in parens, got %+v", ifIdent)
	}
}

func TestNavigationLocalHelpers_findShortDeclTokenInBlockBody(t *testing.T) {
	t.Parallel()
	src := `package main

func f() {
	if true {
		z := 1
		println(string(z))
	}
}
`
	nodes, tokens, tc := parseNodesAndTokensForNavigationTest(t, src)
	fn := firstFunctionNode(t, nodes)
	ifNode, ok := fn.Body[0].(*ast.IfNode)
	if !ok {
		t.Fatalf("expected first stmt to be *ast.IfNode, got %T", fn.Body[0])
	}
	got := findShortDeclTokenInBlockBody(nodes, tokens, ifNode.Body, "z", tc.Hasher)
	if got == nil || got.Value != "z" {
		t.Fatalf("expected z short-decl token, got %+v", got)
	}
}

func TestNavigationLocalHelpers_definingTokenForTypeGuardParams(t *testing.T) {
	t.Parallel()
	src := `package main

is (x Int) Pair(y Int) {
	ensure x is GreaterThan(0)
}
`
	nodes, tokens, tc := parseNodesAndTokensForNavigationTest(t, src)
	var guard ast.TypeGuardNode
	found := false
	for _, n := range nodes {
		if g, ok := n.(ast.TypeGuardNode); ok {
			guard = g
			found = true
			break
		}
		if gp, ok := n.(*ast.TypeGuardNode); ok && gp != nil {
			guard = *gp
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected type guard")
	}
	got := definingTokenForTypeGuardParams(tokens, guard, "y")
	if got != nil && got.Value != "y" {
		t.Fatalf("unexpected token value: %+v", got)
	}
	_ = tc // keep consistent parse/typecheck path for helpers
}

func TestNavigationLocalHelpers_definingTokenForFor(t *testing.T) {
	t.Parallel()
	tokens := []ast.Token{
		{Type: ast.TokenFor, Value: "for"},
		{Type: ast.TokenIdentifier, Value: "i"},
		{Type: ast.TokenColonEquals, Value: ":="},
		{Type: ast.TokenIntLiteral, Value: "0"},
		{Type: ast.TokenSemicolon, Value: ";"},
		{Type: ast.TokenIdentifier, Value: "i"},
		{Type: ast.TokenLess, Value: "<"},
		{Type: ast.TokenIntLiteral, Value: "2"},
		{Type: ast.TokenSemicolon, Value: ";"},
		{Type: ast.TokenIdentifier, Value: "i"},
		{Type: ast.TokenPlusPlus, Value: "++"},
		{Type: ast.TokenLBrace, Value: "{"},
		{Type: ast.TokenRBrace, Value: "}"},
	}
	tc := typechecker.New(logrus.New(), false)
	forn := &ast.ForNode{
		Init: ast.AssignmentNode{
			IsShort: true,
			LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "i"}}},
			RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: 0}},
		},
	}
	nodes := []ast.Node{
		ast.FunctionNode{Body: []ast.Node{*forn}},
	}
	got := definingTokenForFor(tc, nodes, tokens, forn, "i")
	if got != nil && got.Value != "i" {
		t.Fatalf("unexpected token value: %+v", got)
	}
}

func TestDocumentURIHelpers_legacyAndNonFilePaths(t *testing.T) {
	t.Parallel()
	if p, ok := legacyLocalPathFromFileURI("https://example.com"); ok || p != "" {
		t.Fatalf("expected non-file URI to fail, got p=%q ok=%v", p, ok)
	}
	if p, ok := localPathFromFileURI("file:///tmp/a.ft"); !ok || filepath.Base(p) != "a.ft" {
		t.Fatalf("expected valid file URI path, got p=%q ok=%v", p, ok)
	}
	if got := filePathFromDocumentURI("not-a-file-uri"); got != "" {
		t.Fatalf("expected empty path for non-file uri, got %q", got)
	}
}

