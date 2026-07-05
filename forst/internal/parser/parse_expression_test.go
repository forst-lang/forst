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

func TestParseExpressionSource_dottedVariable(t *testing.T) {
	t.Parallel()
	expr, err := ParseExpressionSource(nil, "test", `x.y`)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(ast.VariableNode); !ok {
		t.Fatalf("want VariableNode for x.y, got %T", expr)
	}
}
