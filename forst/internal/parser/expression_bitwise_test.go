package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParse_bitwiseOperators_precedence(t *testing.T) {
	t.Parallel()
	p := NewTestParser("package main\n\nfunc main() { x := 2 << 3 + 4 }\n", ast.SetupTestLogger(nil))
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	fn := nodes[len(nodes)-1].(ast.FunctionNode)
	assign := assertNodeType[ast.AssignmentNode](t, fn.Body[0], "ast.AssignmentNode")
	plus := assertNodeType[ast.BinaryExpressionNode](t, assign.RValues[0], "ast.BinaryExpressionNode")
	if plus.Operator != ast.TokenPlus {
		t.Fatalf("top op: got %s want +", plus.Operator)
	}
	shift := assertNodeType[ast.BinaryExpressionNode](t, plus.Left, "ast.BinaryExpressionNode")
	if shift.Operator != ast.TokenLShift {
		t.Fatalf("shift op: got %s want <<", shift.Operator)
	}
}

func TestParse_bitwiseCompoundAssignment(t *testing.T) {
	t.Parallel()
	src := "package main\n\nfunc main() {\n\tn ^= 1\n\tn <<= 2\n\tn >>= 1\n\tn &^= 3\n}\n"
	p := NewTestParser(src, ast.SetupTestLogger(nil))
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	fn := nodes[len(nodes)-1].(ast.FunctionNode)
	want := []ast.TokenIdent{ast.TokenXorEq, ast.TokenLShiftEq, ast.TokenRShiftEq, ast.TokenAndNotEq}
	for i, op := range want {
		assign := assertNodeType[ast.AssignmentNode](t, fn.Body[i], "ast.AssignmentNode")
		if assign.CompoundOp != op {
			t.Fatalf("stmt %d: got %s want %s", i, assign.CompoundOp, op)
		}
	}
}
