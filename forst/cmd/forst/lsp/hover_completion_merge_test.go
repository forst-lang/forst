package lsp

import (
	"testing"

	"forst/internal/ast"
)

func TestTokensForTypeGuardDocFromPackageMerge_findsGuardTokens(t *testing.T) {
	t.Parallel()
	merge := &packageMergeInfo{
		MemberURIs: []string{"u1", "u2"},
		TokensByURI: map[string][]ast.Token{
			"u1": {
				{Type: ast.TokenIs, Value: "is"},
				{Type: ast.TokenLParen, Value: "("},
				{Type: ast.TokenIdentifier, Value: "x"},
				{Type: ast.TokenIdentifier, Value: "Int"},
				{Type: ast.TokenRParen, Value: ")"},
				{Type: ast.TokenIdentifier, Value: "GuardA"},
			},
			"u2": {
				{Type: ast.TokenIs, Value: "is"},
				{Type: ast.TokenLParen, Value: "("},
				{Type: ast.TokenIdentifier, Value: "y"},
				{Type: ast.TokenIdentifier, Value: "Int"},
				{Type: ast.TokenRParen, Value: ")"},
				{Type: ast.TokenIdentifier, Value: "GuardB"},
			},
		},
	}
	got := tokensForTypeGuardDocFromPackageMerge(merge, "GuardB")
	if len(got) == 0 {
		t.Fatal("expected tokens from merged package for GuardB")
	}
}

func TestMergeLeadingCommentDocShapeField_findsDocAcrossMembers(t *testing.T) {
	t.Parallel()
	merge := &packageMergeInfo{
		MemberURIs: []string{"u1"},
		TokensByURI: map[string][]ast.Token{
			"u1": {
				{Type: ast.TokenType, Value: "type"},
				{Type: ast.TokenIdentifier, Value: "User"},
				{Type: ast.TokenEquals, Value: "="},
				{Type: ast.TokenLBrace, Value: "{"},
				{Type: ast.TokenComment, Value: "// display name"},
				{Type: ast.TokenIdentifier, Value: "name"},
				{Type: ast.TokenColon, Value: ":"},
				{Type: ast.TokenIdentifier, Value: "String"},
				{Type: ast.TokenRBrace, Value: "}"},
			},
		},
	}
	doc := mergeLeadingCommentDocShapeField(merge, nil, "User", "name")
	if doc == "" {
		t.Fatal("expected merged shape field doc for User.name")
	}
}

