package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func testTypeCheckerForCompletionScopeAlgorithms() *typechecker.TypeChecker {
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	return typechecker.New(log, false)
}

func TestMatchIfChainFrom_returnsEnsureBlockFromIfBody(t *testing.T) {
	tc := testTypeCheckerForCompletionScopeAlgorithms()
	baseType := ast.TypeIdent("Int")
	ensureBlock := &ast.EnsureBlockNode{
		Body: []ast.Node{
			ast.AssignmentNode{
				LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier("y")}}},
				RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}},
			},
		},
	}
	ifNode := ast.IfNode{
		Body: []ast.Node{
			ast.EnsureNode{
				Variable:  ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier("x")}},
				Assertion: ast.AssertionNode{BaseType: &baseType},
				Block:     ensureBlock,
			},
		},
	}
	tokens := []ast.Token{
		{Type: ast.TokenIf},
		{Type: ast.TokenLBrace},
		{Type: ast.TokenEnsure},
		{Type: ast.TokenIdentifier, Value: "x"},
		{Type: ast.TokenLBrace},
		{Type: ast.TokenIdentifier, Value: "inside"},
		{Type: ast.TokenRBrace},
		{Type: ast.TokenRBrace},
	}
	scopeNode := matchIfChainFrom(ifNode, tokens, 5, 0, len(tokens), tc)
	if scopeNode == nil {
		t.Fatal("expected ensure block scope node from if body")
	}
	if _, ok := scopeNode.(*ast.EnsureBlockNode); !ok {
		t.Fatalf("expected *ast.EnsureBlockNode, got %T", scopeNode)
	}
}

func TestMatchIfChainFrom_returnsEnsureBlockFromElseIfBody(t *testing.T) {
	tc := testTypeCheckerForCompletionScopeAlgorithms()
	baseType := ast.TypeIdent("Int")
	elseIfEnsureBlock := &ast.EnsureBlockNode{
		Body: []ast.Node{
			ast.AssignmentNode{
				LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier("z")}}},
				RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: 2}},
			},
		},
	}
	ifNode := ast.IfNode{
		Body: []ast.Node{ast.AssignmentNode{}},
		ElseIfs: []ast.ElseIfNode{
			{
				Body: []ast.Node{
					ast.EnsureNode{
						Variable:  ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier("x")}},
						Assertion: ast.AssertionNode{BaseType: &baseType},
						Block:     elseIfEnsureBlock,
					},
				},
			},
		},
	}
	tokens := []ast.Token{
		{Type: ast.TokenIf},
		{Type: ast.TokenLBrace},
		{Type: ast.TokenRBrace},
		{Type: ast.TokenElseIf},
		{Type: ast.TokenLBrace},
		{Type: ast.TokenEnsure},
		{Type: ast.TokenIdentifier, Value: "x"},
		{Type: ast.TokenLBrace},
		{Type: ast.TokenIdentifier, Value: "inside"},
		{Type: ast.TokenRBrace},
		{Type: ast.TokenRBrace},
	}
	scopeNode := matchIfChainFrom(ifNode, tokens, 8, 0, len(tokens), tc)
	if scopeNode == nil {
		t.Fatal("expected ensure block scope node from else-if body")
	}
	if _, ok := scopeNode.(*ast.EnsureBlockNode); !ok {
		t.Fatalf("expected *ast.EnsureBlockNode, got %T", scopeNode)
	}
}

func TestMatchIfChainFrom_returnsEnsureBlockFromElseBody(t *testing.T) {
	tc := testTypeCheckerForCompletionScopeAlgorithms()
	baseType := ast.TypeIdent("Int")
	elseEnsureBlock := &ast.EnsureBlockNode{
		Body: []ast.Node{
			ast.AssignmentNode{
				LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier("w")}}},
				RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: 3}},
			},
		},
	}
	ifNode := ast.IfNode{
		Body: []ast.Node{ast.AssignmentNode{}},
		Else: &ast.ElseBlockNode{
			Body: []ast.Node{
				ast.EnsureNode{
					Variable:  ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier("x")}},
					Assertion: ast.AssertionNode{BaseType: &baseType},
					Block:     elseEnsureBlock,
				},
			},
		},
	}
	tokens := []ast.Token{
		{Type: ast.TokenIf},
		{Type: ast.TokenLBrace},
		{Type: ast.TokenRBrace},
		{Type: ast.TokenElse},
		{Type: ast.TokenLBrace},
		{Type: ast.TokenEnsure},
		{Type: ast.TokenIdentifier, Value: "x"},
		{Type: ast.TokenLBrace},
		{Type: ast.TokenIdentifier, Value: "inside"},
		{Type: ast.TokenRBrace},
		{Type: ast.TokenRBrace},
	}
	scopeNode := matchIfChainFrom(ifNode, tokens, 8, 0, len(tokens), tc)
	if scopeNode == nil {
		t.Fatal("expected ensure block scope node from else body")
	}
	if _, ok := scopeNode.(*ast.EnsureBlockNode); !ok {
		t.Fatalf("expected *ast.EnsureBlockNode, got %T", scopeNode)
	}
}

func TestTypeGuardBodyBraces_andScopeForPosition(t *testing.T) {
	tokens := lexTokensForLSPHelperTest(`package main

type P = String

is (p P) Strong {
  ensure p is Min(3)
}
`)
	leftBrace, rightBrace := typeGuardBodyBraces(tokens, "Strong")
	if leftBrace < 0 || rightBrace <= leftBrace {
		t.Fatalf("expected valid type-guard body braces, got l=%d r=%d", leftBrace, rightBrace)
	}
	missingLeft, missingRight := typeGuardBodyBraces(tokens, "Missing")
	if missingLeft != -1 || missingRight != -1 {
		t.Fatalf("expected missing type-guard braces to be -1/-1, got l=%d r=%d", missingLeft, missingRight)
	}

	log := logrus.New()
	server := NewLSPServer("8080", log)
	dir := t.TempDir()
	filePath := filepath.Join(dir, "guard_scope.ft")
	source := `package main

type P = String

is (p P) Strong {
  ensure p is Min(3)
}
`
	if err := os.WriteFile(filePath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, filePath)
	server.documentMu.Lock()
	server.openDocuments[uri] = source
	server.documentMu.Unlock()

	context, ok := server.analyzeForstDocument(uri)
	if !ok || context == nil || context.ParseErr != nil {
		t.Fatalf("analyze: ok=%v parseErr=%v", ok, context.ParseErr)
	}
	if context.TC == nil {
		t.Fatal("expected non-nil typechecker")
	}

	var guardNode ast.TypeGuardNode
	guardFound := false
	for _, node := range context.Nodes {
		switch typedNode := node.(type) {
		case ast.TypeGuardNode:
			if string(typedNode.Ident) == "Strong" {
				guardNode = typedNode
				guardFound = true
			}
		case *ast.TypeGuardNode:
			if typedNode != nil && string(typedNode.Ident) == "Strong" {
				guardNode = *typedNode
				guardFound = true
			}
		}
	}
	if !guardFound {
		t.Fatal("expected to find Strong type guard node")
	}

	scopeNodeInside := typeGuardScopeForPosition(guardNode, context.Tokens, leftBrace+1, context.TC)
	if scopeNodeInside == nil {
		t.Fatal("expected type guard scope node for position inside body")
	}
	outsideScope := typeGuardScopeForPosition(guardNode, context.Tokens, 0, context.TC)
	if outsideScope != nil {
		t.Fatalf("expected nil scope outside type guard body, got %T", outsideScope)
	}
}
