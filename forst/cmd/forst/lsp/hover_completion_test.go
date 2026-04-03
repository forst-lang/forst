package lsp

import (
	"testing"

	"forst/internal/ast"
)

func TestStripCommentBody(t *testing.T) {
	t.Parallel()
	if got := stripCommentBody("//  hello "); got != "hello" {
		t.Fatalf("got %q", got)
	}
	if got := stripCommentBody("/* x */"); got != "x" {
		t.Fatalf("got %q", got)
	}
}

func TestLeadingCommentDocBeforeFunc(t *testing.T) {
	t.Parallel()
	tokens := []ast.Token{
		{Type: ast.TokenPackage, Value: "package", Line: 1, Column: 1},
		{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 9},
		{Type: ast.TokenComment, Value: "// doc line 1", Line: 3, Column: 1},
		{Type: ast.TokenComment, Value: "// doc line 2", Line: 4, Column: 1},
		{Type: ast.TokenFunc, Value: "func", Line: 5, Column: 1},
		{Type: ast.TokenIdentifier, Value: "Foo", Line: 5, Column: 6},
	}
	got := leadingCommentDocBeforeFunc(tokens, "Foo")
	want := "doc line 1\ndoc line 2"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFindFuncKeywordIndex(t *testing.T) {
	t.Parallel()
	tokens := []ast.Token{
		{Type: ast.TokenFunc, Value: "func"},
		{Type: ast.TokenIdentifier, Value: "a"},
		{Type: ast.TokenFunc, Value: "func"},
		{Type: ast.TokenIdentifier, Value: "b"},
	}
	if i := findFuncKeywordIndex(tokens, "b"); i != 2 {
		t.Fatalf("got index %d", i)
	}
}
