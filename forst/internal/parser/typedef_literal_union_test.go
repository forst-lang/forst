package parser

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestParseTypeDefExpr_stringLiteralUnion(t *testing.T) {
	t.Parallel()
	src := `package main

type TaskStatus = "todo" | "in_progress" | "success" | "failed"

func main() {}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	var td *ast.TypeDefNode
	for _, n := range nodes {
		if d, ok := n.(ast.TypeDefNode); ok && d.Ident == "TaskStatus" {
			td = &d
			break
		}
	}
	if td == nil {
		t.Fatal("expected TaskStatus")
	}
	bin, ok := td.Expr.(ast.TypeDefBinaryExpr)
	if !ok {
		t.Fatalf("expected TypeDefBinaryExpr, got %T", td.Expr)
	}
	if !bin.IsDisjunction() {
		t.Fatal("expected |")
	}
}

func TestParseTypeDefExpr_rejectsFloatLiteralUnion(t *testing.T) {
	t.Parallel()
	src := `package main
type Bad = 1.5 | 2.5
`
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		msg := ""
		switch e := r.(type) {
		case error:
			msg = e.Error()
		case string:
			msg = e
		default:
			msg = "recovered"
		}
		if !strings.Contains(msg, "refinement-unsupported-union") {
			t.Fatalf("want unsupported-union diagnostic, got %v", r)
		}
	}()
	_, _ = NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
}
