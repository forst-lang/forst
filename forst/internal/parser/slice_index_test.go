package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParseExpression_sliceSubscriptOnVariable(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
	xs := ["a", "b", "c"]
	s := xs[1]
}
`
	logger := ast.SetupTestLogger(nil)
	p := NewTestParser(src, logger)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	fn := findFunction(t, nodes, "main")
	assign := fn.Body[1].(ast.AssignmentNode)
	rhs := assign.RValues[0].(ast.IndexExpressionNode)
	if _, ok := rhs.Target.(ast.VariableNode); !ok {
		t.Fatalf("target: want VariableNode, got %T", rhs.Target)
	}
	if lit, ok := rhs.Index.(ast.IntLiteralNode); !ok || lit.Value != 1 {
		t.Fatalf("index: want IntLiteral 1, got %#v", rhs.Index)
	}
}

func TestParseAssignment_indexedLValue(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
	xs := ["a", "b"]
	xs[0] = "z"
}
`
	logger := ast.SetupTestLogger(nil)
	p := NewTestParser(src, logger)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	fn := findFunction(t, nodes, "main")
	assign := fn.Body[1].(ast.AssignmentNode)
	lv, ok := assign.LValues[0].(ast.IndexExpressionNode)
	if !ok {
		t.Fatalf("lhs: want IndexExpressionNode, got %T", assign.LValues[0])
	}
	if _, ok := lv.Target.(ast.VariableNode); !ok {
		t.Fatalf("lhs target: want VariableNode, got %T", lv.Target)
	}
	if lit, ok := lv.Index.(ast.IntLiteralNode); !ok || lit.Value != 0 {
		t.Fatalf("lhs index: want 0, got %#v", lv.Index)
	}
}
