package typechecker

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestInferForNode_rangeOverIntSlice(t *testing.T) {
	tc := New(logrus.New(), false)
	fn := ast.FunctionNode{
		Ident: ast.Ident{ID: "main"},
		Body: []ast.Node{
			ast.AssignmentNode{
				LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "xs"}}},
				RValues: []ast.ExpressionNode{
					ast.ArrayLiteralNode{
						Value: []ast.LiteralNode{ast.IntLiteralNode{Value: 10}},
						Type:  ast.TypeNode{Ident: ast.TypeImplicit},
					},
				},
				IsShort: true,
			},
			&ast.ForNode{
				IsRange:    true,
				RangeX:     ast.VariableNode{Ident: ast.Ident{ID: "xs"}},
				RangeKey:   &ast.Ident{ID: "i"},
				RangeValue: &ast.Ident{ID: "v"},
				RangeShort: true,
				Body:       []ast.Node{},
			},
		},
	}
	if err := tc.CheckTypes([]ast.Node{fn}); err != nil {
		t.Fatal(err)
	}
}

func TestInferForNode_classicForWithPostIncrement(t *testing.T) {
	tc := New(logrus.New(), false)
	fn := ast.FunctionNode{
		Ident: ast.Ident{ID: "main"},
		Body: []ast.Node{
			&ast.ForNode{
				Init: ast.AssignmentNode{
					LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "i"}}},
					RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: 0}},
					IsShort: true,
				},
				Cond: ast.BinaryExpressionNode{
					Left:     ast.VariableNode{Ident: ast.Ident{ID: "i"}},
					Operator: ast.TokenLess,
					Right:    ast.IntLiteralNode{Value: 2},
				},
				Post: ast.UnaryExpressionNode{
					Operator: ast.TokenPlusPlus,
					Operand:  ast.VariableNode{Ident: ast.Ident{ID: "i"}},
				},
				Body: []ast.Node{},
			},
		},
	}
	if err := tc.CheckTypes([]ast.Node{fn}); err != nil {
		t.Fatal(err)
	}
}

func TestInferForNode_breakInsideLoop(t *testing.T) {
	tc := New(logrus.New(), false)
	fn := ast.FunctionNode{
		Ident: ast.Ident{ID: "main"},
		Body: []ast.Node{
			&ast.ForNode{
				Body: []ast.Node{&ast.BreakNode{}},
			},
		},
	}
	if err := tc.CheckTypes([]ast.Node{fn}); err != nil {
		t.Fatal(err)
	}
}

func TestInferBreakOutsideLoop(t *testing.T) {
	tc := New(logrus.New(), false)
	fn := ast.FunctionNode{
		Ident: ast.Ident{ID: "main"},
		Body:  []ast.Node{&ast.BreakNode{}},
	}
	err := tc.CheckTypes([]ast.Node{fn})
	if err == nil {
		t.Fatal("expected break outside loop to fail")
	}
}
