package parser

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestParseBlock_emptyBody(t *testing.T) {
	t.Parallel()
	src := `package main

func empty() {
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	fn := assertNodeType[ast.FunctionNode](t, nodes[1], "ast.FunctionNode")
	if len(fn.Body) != 0 {
		t.Fatalf("expected empty block, got %d statements", len(fn.Body))
	}
}

func TestParseBlock_leadingComment(t *testing.T) {
	t.Parallel()
	src := `package main

func withComment() {
	// keep
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	fn := assertNodeType[ast.FunctionNode](t, nodes[1], "ast.FunctionNode")
	if len(fn.Body) != 1 {
		t.Fatalf("expected comment statement, got %d nodes", len(fn.Body))
	}
	if _, ok := fn.Body[0].(ast.CommentNode); !ok {
		t.Fatalf("expected CommentNode, got %T", fn.Body[0])
	}
}

func TestParseBlock_missingClosingBrace_reportsError(t *testing.T) {
	t.Parallel()
	src := `package main

func broken() {
	x := 1
`
	err := parseShouldFail(src)
	if err == nil || !strings.Contains(err.Error(), "Parse error") {
		t.Fatalf("err = %v", err)
	}
}
