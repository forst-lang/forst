package typechecker

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestInferIfStatement_withInitAndCondition(t *testing.T) {
	tc := New(logrus.New(), false)
	fn := ast.FunctionNode{
		Ident: ast.Ident{ID: "main"},
		Body: []ast.Node{
			&ast.IfNode{
				Init: ast.AssignmentNode{
					LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "x"}}},
					RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}},
					IsShort: true,
				},
				Condition: ast.BinaryExpressionNode{
					Left:     ast.VariableNode{Ident: ast.Ident{ID: "x"}},
					Operator: ast.TokenGreater,
					Right:    ast.IntLiteralNode{Value: 0},
				},
				Body: []ast.Node{},
			},
		},
	}
	if err := tc.CheckTypes([]ast.Node{fn}); err != nil {
		t.Fatal(err)
	}
}
