package parser //nolint:revive // package name matches internal/parser; overlaps with text/template parser etc.

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

func TestParseExpression_sliceSubsliceForms(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		expr    string
		wantLow *int64
		wantHigh *int64
	}{
		{name: "low high", expr: "xs[1:3]", wantLow: int64Ptr(1), wantHigh: int64Ptr(3)},
		{name: "low only", expr: "xs[1:]", wantLow: int64Ptr(1)},
		{name: "high only", expr: "xs[:3]", wantHigh: int64Ptr(3)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := "package main\nfunc main() { _ = " + tc.expr + " }\n"
			logger := ast.SetupTestLogger(nil)
			p := NewTestParser(src, logger)
			nodes, err := p.ParseFile()
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			fn := findFunction(t, nodes, "main")
			assign := fn.Body[0].(ast.AssignmentNode)
			slice, ok := assign.RValues[0].(ast.SliceExpressionNode)
			if !ok {
				t.Fatalf("rhs: want SliceExpressionNode, got %T", assign.RValues[0])
			}
			if _, ok := slice.Target.(ast.VariableNode); !ok {
				t.Fatalf("target: want VariableNode xs, got %T", slice.Target)
			}
			if tc.wantLow != nil {
				lit, ok := slice.Low.(ast.IntLiteralNode)
				if !ok || lit.Value != *tc.wantLow {
					t.Fatalf("low: want %d, got %#v", *tc.wantLow, slice.Low)
				}
			} else if slice.Low != nil {
				t.Fatalf("low: want nil, got %#v", slice.Low)
			}
			if tc.wantHigh != nil {
				lit, ok := slice.High.(ast.IntLiteralNode)
				if !ok || lit.Value != *tc.wantHigh {
					t.Fatalf("high: want %d, got %#v", *tc.wantHigh, slice.High)
				}
			} else if slice.High != nil {
				t.Fatalf("high: want nil, got %#v", slice.High)
			}
		})
	}
}

func int64Ptr(v int64) *int64 { return &v }
