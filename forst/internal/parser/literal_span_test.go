package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParseLiterals_setSourceSpans(t *testing.T) {
	t.Parallel()
	src := `package main
func f() {
	_ = 42
	_ = "hi"
	_ = true
	_ = nil
	_ = [1, 2]
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	var foundInt, foundStr, foundBool, foundNil, foundArr bool
	var walk func(n ast.Node)
	walk = func(n ast.Node) {
		if n == nil {
			return
		}
		switch v := n.(type) {
		case ast.IntLiteralNode:
			if !v.Span.IsSet() {
				t.Fatalf("IntLiteralNode missing span: %+v", v)
			}
			foundInt = true
		case ast.StringLiteralNode:
			if !v.Span.IsSet() {
				t.Fatalf("StringLiteralNode missing span: %+v", v)
			}
			foundStr = true
		case ast.BoolLiteralNode:
			if !v.Span.IsSet() {
				t.Fatalf("BoolLiteralNode missing span: %+v", v)
			}
			foundBool = true
		case ast.NilLiteralNode:
			if !v.Span.IsSet() {
				t.Fatalf("NilLiteralNode missing span: %+v", v)
			}
			foundNil = true
		case ast.ArrayLiteralNode:
			if !v.Span.IsSet() {
				t.Fatalf("ArrayLiteralNode missing span: %+v", v)
			}
			foundArr = true
			for _, el := range v.Value {
				walk(el)
			}
		case ast.FunctionNode:
			for _, stmt := range v.Body {
				walk(stmt)
			}
		case ast.AssignmentNode:
			for _, rv := range v.RValues {
				walk(rv)
			}
		}
	}
	for _, n := range nodes {
		walk(n)
	}
	if !foundInt || !foundStr || !foundBool || !foundNil || !foundArr {
		t.Fatalf("missing literals: int=%v str=%v bool=%v nil=%v arr=%v", foundInt, foundStr, foundBool, foundNil, foundArr)
	}
}
