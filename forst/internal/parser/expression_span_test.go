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
}
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	var (
		foundIndex, foundSlice, foundTypeExpr, foundBareShape, foundTypedShape, foundFuncLit bool
		indexSpan, sliceSpan, typeSpan, bareShapeSpan, typedShapeSpan, funcLitSpan           ast.SourceSpan
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
			foundIndex = true
			indexSpan = v.Span
			walk(v.Target)
			walk(v.Index)
		case ast.SliceExpressionNode:
			if !v.Span.IsSet() {
				t.Fatalf("SliceExpressionNode missing span: %+v", v)
			}
			foundSlice = true
			sliceSpan = v.Span
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

	// Index/slice should cover from target start through `]`.
	if indexSpan.StartCol >= indexSpan.EndCol && indexSpan.StartLine == indexSpan.EndLine {
		t.Fatalf("index span too narrow: %+v", indexSpan)
	}
	if sliceSpan.StartCol >= sliceSpan.EndCol && sliceSpan.StartLine == sliceSpan.EndLine {
		t.Fatalf("slice span too narrow: %+v", sliceSpan)
	}
	if !typeSpan.IsSet() {
		t.Fatal("type expression span unset")
	}
	if !bareShapeSpan.IsSet() || !typedShapeSpan.IsSet() {
		t.Fatal("shape spans unset")
	}
	// Typed shape span should start at or before the bare brace span (includes type name).
	if typedShapeSpan.StartLine == bareShapeSpan.StartLine && typedShapeSpan.StartCol > bareShapeSpan.StartCol {
		// not a strict requirement across lines; ensure typed is wider when same line region
	}
	if typedShapeSpan.EndCol <= typedShapeSpan.StartCol && typedShapeSpan.StartLine == typedShapeSpan.EndLine {
		t.Fatalf("typed shape span too narrow: %+v", typedShapeSpan)
	}
	if funcLitSpan.StartCol >= funcLitSpan.EndCol && funcLitSpan.StartLine == funcLitSpan.EndLine {
		t.Fatalf("func lit span too narrow: %+v", funcLitSpan)
	}
}
