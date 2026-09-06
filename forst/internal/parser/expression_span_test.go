package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParseExpressionSpans_indexSliceTypeShapeFuncLit(t *testing.T) {
	t.Parallel()
	src := `package main
type User = { name: String }
func f() {
	xs := [1, 2, 3]
	_ = xs[1]
	_ = xs[1:3]
	_ = make(Array(Int), 0)
	_ = { name: "a" }
	_ = User{ name: "b" }
	_ = func(): Int { return 1 }
	_ = xs[1][0]
	_ = xs[1:3][:1]
	_ = Ok(1)
	_ = Err("e")
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	var (
		indexSpan, sliceSpan, typeSpan, bareShapeSpan, typedShapeSpan, funcLitSpan ast.SourceSpan
		nestedIndexSpan, nestedSliceSpan, okSpan, errSpan                          ast.SourceSpan
		foundIndex, foundSlice, foundTypeExpr, foundBareShape, foundTypedShape     bool
		foundFuncLit, foundNestedIndex, foundNestedSlice, foundOk, foundErr        bool
	)

	var walk func(n ast.Node)
	walk = func(n ast.Node) {
		if n == nil {
			return
		}
		switch v := n.(type) {
		case ast.IndexExpressionNode:
			if !v.Span.IsSet() {
				t.Fatalf("IndexExpressionNode missing span: %+v", v)
			}
			if _, ok := v.Target.(ast.IndexExpressionNode); ok {
				foundNestedIndex = true
				nestedIndexSpan = v.Span
			} else if !foundIndex {
				foundIndex = true
				indexSpan = v.Span
			}
			walk(v.Target)
			walk(v.Index)
		case ast.SliceExpressionNode:
			if !v.Span.IsSet() {
				t.Fatalf("SliceExpressionNode missing span: %+v", v)
			}
			if _, ok := v.Target.(ast.SliceExpressionNode); ok {
				foundNestedSlice = true
				nestedSliceSpan = v.Span
			} else if !foundSlice {
				foundSlice = true
				sliceSpan = v.Span
			}
			walk(v.Target)
			if v.Low != nil {
				walk(v.Low)
			}
			if v.High != nil {
				walk(v.High)
			}
		case ast.TypeExpressionNode:
			if !v.Span.IsSet() {
				t.Fatalf("TypeExpressionNode missing span: %+v", v)
			}
			foundTypeExpr = true
			typeSpan = v.Span
		case ast.ShapeNode:
			if !v.Span.IsSet() {
				t.Fatalf("ShapeNode missing span: %+v", v)
			}
			if v.BaseType != nil && string(*v.BaseType) == "User" {
				foundTypedShape = true
				typedShapeSpan = v.Span
			} else if v.BaseType == nil {
				foundBareShape = true
				bareShapeSpan = v.Span
			}
		case ast.FunctionLiteralNode:
			if !v.Span.IsSet() {
				t.Fatalf("FunctionLiteralNode missing span: %+v", v)
			}
			foundFuncLit = true
			funcLitSpan = v.Span
			for _, stmt := range v.Body {
				walk(stmt)
			}
		case ast.OkExprNode:
			if !v.Span.IsSet() {
				t.Fatalf("OkExprNode missing span: %+v", v)
			}
			foundOk = true
			okSpan = v.Span
			walk(v.Value)
		case ast.ErrExprNode:
			if !v.Span.IsSet() {
				t.Fatalf("ErrExprNode missing span: %+v", v)
			}
			foundErr = true
			errSpan = v.Span
			walk(v.Value)
		case ast.FunctionCallNode:
			for _, arg := range v.Arguments {
				walk(arg)
			}
		case ast.FunctionNode:
			for _, stmt := range v.Body {
				walk(stmt)
			}
		case ast.AssignmentNode:
			for _, rv := range v.RValues {
				walk(rv)
			}
		case ast.TypeDefNode:
			// skip typedef shapes for bare/typed literal checks
		}
	}
	for _, n := range nodes {
		walk(n)
	}

	if !foundIndex || !foundSlice || !foundTypeExpr || !foundBareShape || !foundTypedShape || !foundFuncLit {
		t.Fatalf("missing nodes: index=%v slice=%v type=%v bareShape=%v typedShape=%v funcLit=%v",
			foundIndex, foundSlice, foundTypeExpr, foundBareShape, foundTypedShape, foundFuncLit)
	}
	if !foundNestedIndex || !foundNestedSlice || !foundOk || !foundErr {
		t.Fatalf("missing extra nodes: nestedIndex=%v nestedSlice=%v ok=%v err=%v",
			foundNestedIndex, foundNestedSlice, foundOk, foundErr)
	}

	cases := []struct {
		name string
		got  ast.SourceSpan
		want ast.SourceSpan
	}{
		{name: "index", got: indexSpan, want: ast.SourceSpan{StartLine: 5, StartCol: 6, EndLine: 5, EndCol: 11}},
		{name: "slice", got: sliceSpan, want: ast.SourceSpan{StartLine: 6, StartCol: 6, EndLine: 6, EndCol: 13}},
		{name: "typeExpr", got: typeSpan, want: ast.SourceSpan{StartLine: 7, StartCol: 11, EndLine: 7, EndCol: 21}},
		{name: "bareShape", got: bareShapeSpan, want: ast.SourceSpan{StartLine: 8, StartCol: 6, EndLine: 8, EndCol: 19}},
		{name: "typedShapeStartsAtUser", got: typedShapeSpan, want: ast.SourceSpan{StartLine: 9, StartCol: 6, EndLine: 9, EndCol: 23}},
		{name: "funcLit", got: funcLitSpan, want: ast.SourceSpan{StartLine: 10, StartCol: 6, EndLine: 10, EndCol: 30}},
		{name: "nestedIndex", got: nestedIndexSpan, want: ast.SourceSpan{StartLine: 11, StartCol: 6, EndLine: 11, EndCol: 14}},
		{name: "nestedSlice", got: nestedSliceSpan, want: ast.SourceSpan{StartLine: 12, StartCol: 6, EndLine: 12, EndCol: 17}},
		{name: "okExprStartsAtOk", got: okSpan, want: ast.SourceSpan{StartLine: 13, StartCol: 6, EndLine: 13, EndCol: 11}},
		{name: "errExprStartsAtErr", got: errSpan, want: ast.SourceSpan{StartLine: 14, StartCol: 6, EndLine: 14, EndCol: 14}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("got %+v want %+v", tc.got, tc.want)
			}
		})
	}
}
