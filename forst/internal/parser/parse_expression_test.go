package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParseExpressionSource_qualifiedCall(t *testing.T) {
	t.Parallel()
	expr, err := ParseExpressionSource(nil, "test", `time.Now()`)
	if err != nil {
		t.Fatal(err)
	}
	fc, ok := expr.(ast.FunctionCallNode)
	if !ok {
		t.Fatalf("want FunctionCallNode, got %T", expr)
	}
	if fc.Function.ID != "time.Now" {
		t.Fatalf("function id = %q", fc.Function.ID)
	}
}

func TestParseExpressionSource_qualifiedFieldPath(t *testing.T) {
	t.Parallel()
	expr, err := ParseExpressionSource(nil, "test", `x.y`)
	if err != nil {
		t.Fatal(err)
	}
	vn, ok := expr.(ast.VariableNode)
	if !ok {
		t.Fatalf("want VariableNode for x.y, got %T", expr)
	}
	if vn.Ident.ID != "x.y" {
		t.Fatalf("ident id = %q, want x.y", vn.Ident.ID)
	}
}

func TestParseExpressionSource_qualifiedFieldPathOnLocal(t *testing.T) {
	t.Parallel()
	expr, err := ParseExpressionSource(nil, "test", `cmd.ProcessState`)
	if err != nil {
		t.Fatal(err)
	}
	vn, ok := expr.(ast.VariableNode)
	if !ok {
		t.Fatalf("want VariableNode, got %T", expr)
	}
	if vn.Ident.ID != "cmd.ProcessState" {
		t.Fatalf("ident id = %q", vn.Ident.ID)
	}
}

func TestParseExpressionSource_subslice(t *testing.T) {
	t.Parallel()
	expr, err := ParseExpressionSource(nil, "test", `argv[1:]`)
	if err != nil {
		t.Fatal(err)
	}
	slice, ok := expr.(ast.SliceExpressionNode)
	if !ok {
		t.Fatalf("want SliceExpressionNode, got %T", expr)
	}
	if _, ok := slice.Target.(ast.VariableNode); !ok {
		t.Fatalf("target: want VariableNode, got %T", slice.Target)
	}
	lit, ok := slice.Low.(ast.IntLiteralNode)
	if !ok || lit.Value != 1 {
		t.Fatalf("low: want 1, got %#v", slice.Low)
	}
	if slice.High != nil {
		t.Fatalf("high: want nil, got %#v", slice.High)
	}
}
