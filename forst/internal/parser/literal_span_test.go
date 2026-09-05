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
	var intSpan, strSpan, boolSpan, nilSpan, arrSpan ast.SourceSpan
	var foundInt, foundStr, foundBool, foundNil, foundArr bool
	var walk func(n ast.Node)
	walk = func(n ast.Node) {
		if n == nil {
			return
		}
		switch v := n.(type) {
		case ast.IntLiteralNode:
			if v.Value != 42 {
				return
			}
			if !v.Span.IsSet() {
				t.Fatalf("IntLiteralNode missing span: %+v", v)
			}
			foundInt = true
			intSpan = v.Span
		case ast.StringLiteralNode:
			if !v.Span.IsSet() {
				t.Fatalf("StringLiteralNode missing span: %+v", v)
			}
			foundStr = true
			strSpan = v.Span
		case ast.BoolLiteralNode:
			if !v.Span.IsSet() {
				t.Fatalf("BoolLiteralNode missing span: %+v", v)
			}
			foundBool = true
			boolSpan = v.Span
		case ast.NilLiteralNode:
			if !v.Span.IsSet() {
				t.Fatalf("NilLiteralNode missing span: %+v", v)
			}
			foundNil = true
			nilSpan = v.Span
		case ast.ArrayLiteralNode:
			if !v.Span.IsSet() {
				t.Fatalf("ArrayLiteralNode missing span: %+v", v)
			}
			foundArr = true
			arrSpan = v.Span
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

	cases := []struct {
		name string
		got  ast.SourceSpan
		want ast.SourceSpan
	}{
		{name: "int", got: intSpan, want: ast.SourceSpan{StartLine: 3, StartCol: 6, EndLine: 3, EndCol: 8}},
		{name: "string", got: strSpan, want: ast.SourceSpan{StartLine: 4, StartCol: 6, EndLine: 4, EndCol: 10}},
		{name: "bool", got: boolSpan, want: ast.SourceSpan{StartLine: 5, StartCol: 6, EndLine: 5, EndCol: 10}},
		{name: "nil", got: nilSpan, want: ast.SourceSpan{StartLine: 6, StartCol: 6, EndLine: 6, EndCol: 9}},
		{name: "array", got: arrSpan, want: ast.SourceSpan{StartLine: 7, StartCol: 6, EndLine: 7, EndCol: 12}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("got %+v want %+v", tc.got, tc.want)
			}
		})
	}
}
