package hasher

import (
	"testing"

	"forst/internal/ast"
)

// TestHashNode_exhaustiveSwitchCases hits remaining hashUncached switch arms (value + pointer).
func TestHashNode_exhaustiveSwitchCases(t *testing.T) {
	t.Parallel()
	h := New()
	baseStr := ast.TypeString
	intType := ast.TypeNode{Ident: ast.TypeInt}

	tests := []struct {
		name string
		node ast.Node
	}{
		{"TypeDefBinaryExpr", ast.TypeDefBinaryExpr{
			Op: ast.TokenBitwiseOr,
			Left: ast.TypeDefShapeExpr{Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}},
			Right: ast.TypeDefShapeExpr{Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{
				"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
			}}},
		}},
		{"FunctionLiteralNode", ast.FunctionLiteralNode{
			Params: []ast.ParamNode{ast.SimpleParamNode{Ident: ast.Ident{ID: "x"}, Type: intType}},
			Body:   []ast.Node{ast.ReturnNode{Values: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}}}},
		}},
		{"FunctionCallNode_callee", ast.FunctionCallNode{
			Callee:    ast.FunctionLiteralNode{Body: []ast.Node{}},
			Arguments: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}},
		}},
		{"FunctionCallNode_ptr", &ast.FunctionCallNode{
			Function: ast.Ident{ID: "f"}, Arguments: []ast.ExpressionNode{},
		}},
		{"MethodCallNode", ast.MethodCallNode{
			Receiver: ast.VariableNode{Ident: ast.Ident{ID: "r"}},
			Method:   ast.Ident{ID: "M"},
			Arguments: []ast.ExpressionNode{
				ast.StringLiteralNode{Value: "a"},
			},
		}},
		{"MethodCallNode_ptr", &ast.MethodCallNode{
			Receiver: ast.VariableNode{Ident: ast.Ident{ID: "r"}}, Method: ast.Ident{ID: "M"},
		}},
		{"IndexExpressionNode", ast.IndexExpressionNode{
			Target: ast.VariableNode{Ident: ast.Ident{ID: "xs"}}, Index: ast.IntLiteralNode{Value: 0},
		}},
		{"SliceExpressionNode", ast.SliceExpressionNode{Target: ast.VariableNode{Ident: ast.Ident{ID: "s"}}}},
		{"SpreadExpressionNode", ast.SpreadExpressionNode{Expr: ast.VariableNode{Ident: ast.Ident{ID: "a"}}}},
		{"FieldAccessNode", ast.FieldAccessNode{
			Target: ast.VariableNode{Ident: ast.Ident{ID: "o"}}, Field: ast.Ident{ID: "f"},
		}},
		{"TypeExpressionNode", ast.TypeExpressionNode{Type: intType}},
		{"ConstGroupNode", ast.ConstGroupNode{
			Specs: []ast.ConstSpec{
				{Name: ast.Ident{ID: "A"}, Value: ast.IntLiteralNode{Value: 1}},
				{Name: ast.Ident{ID: "B"}, Value: ast.IotaLiteralNode{}},
			},
		}},
		{"IotaLiteralNode", ast.IotaLiteralNode{}},
		{"ReferenceNode", ast.ReferenceNode{Value: ast.VariableNode{Ident: ast.Ident{ID: "p"}}}},
		{"ReferenceNode_ptr", &ast.ReferenceNode{Value: ast.VariableNode{Ident: ast.Ident{ID: "p"}}}},
		{"UseNode", ast.UseNode{
			Ident: &ast.Ident{ID: "ctx"}, ContractType: ast.TypeNode{Ident: "Context"},
		}},
		{"WithNode", ast.WithNode{
			Wiring: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}},
			Body:   []ast.Node{ast.IntLiteralNode{Value: 1}},
		}},
		{"ArrayLiteralNode", ast.ArrayLiteralNode{
			Type:  intType,
			Value: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}, ast.IntLiteralNode{Value: 2}},
		}},
		{"ArrayLiteralNode_ptr", &ast.ArrayLiteralNode{Value: []ast.ExpressionNode{ast.IntLiteralNode{Value: 3}}}},
		{"DereferenceNode", ast.DereferenceNode{Value: ast.VariableNode{Ident: ast.Ident{ID: "p"}}}},
		{"NilLiteralNode", ast.NilLiteralNode{}},
		{"SwitchNode", ast.SwitchNode{
			Init: ast.AssignmentNode{
				IsShort: true,
				LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "x"}}},
				RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}},
			},
			Tag: ast.VariableNode{Ident: ast.Ident{ID: "x"}},
			Clauses: []ast.SwitchClauseNode{
				{Values: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}}, Body: []ast.Node{ast.FallthroughNode{}}},
			},
		}},
		{"SwitchNode_ptr", func() ast.Node {
			sw := ast.SwitchNode{
				Tag: ast.VariableNode{Ident: ast.Ident{ID: "x"}},
				Clauses: []ast.SwitchClauseNode{
					{Body: []ast.Node{&ast.GotoNode{Label: &ast.Ident{ID: "L"}}}},
				},
			}
			return &sw
		}()},
		{"FallthroughNode", ast.FallthroughNode{}},
		{"GotoNode", &ast.GotoNode{Label: &ast.Ident{ID: "loop"}}},
		{"LabeledStmtNode", &ast.LabeledStmtNode{
			Label: &ast.Ident{ID: "L"},
			Stmt:  ast.ReturnNode{Values: []ast.ExpressionNode{ast.IntLiteralNode{Value: 0}}},
		}},
		{"FloatLiteralNode_ptr", &ast.FloatLiteralNode{Value: 1.5}},
		{"RuneLiteralNode_ptr", &ast.RuneLiteralNode{Value: int64('x')}},
		{"StringLiteralNode_ptr", &ast.StringLiteralNode{Value: "s"}},
		{"BoolLiteralNode_ptr", &ast.BoolLiteralNode{Value: true}},
		{"IntLiteralNode_ptr", &ast.IntLiteralNode{Value: 7}},
		{"DestructuredParamNode", ast.DestructuredParamNode{
			Fields: []string{"a", "b"}, Type: intType,
		}},
		{"EnsureBlockNode_ptr", &ast.EnsureBlockNode{Body: []ast.Node{ast.IntLiteralNode{Value: 2}}}},
		{"ShapeNode_full", ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{
			"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
			"nested": {Shape: &ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{
				"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
			}}},
		}}},
		{"ShapeNode_ptr", &ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{
			"k": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
		}}},
		{"ShapeFieldNode_shape", ast.ShapeFieldNode{Shape: &ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}}},
		{"AssertionNode_full", ast.AssertionNode{
			BaseType: &baseStr,
			Constraints: []ast.ConstraintNode{
				{Name: "Max", Args: []ast.ConstraintArgumentNode{{Shape: &ast.ShapeNode{}}}},
			},
		}},
		{"ConstraintNode", ast.ConstraintNode{Name: "NonEmpty"}},
		{"TypeDefShapeExpr", ast.TypeDefShapeExpr{Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{
			"id": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
		}}}},
		{"FunctionNode_receiver", ast.FunctionNode{
			Ident: ast.Ident{ID: "meth"},
			Receiver: &ast.SimpleParamNode{
				Ident: ast.Ident{ID: "s"}, Type: ast.TypeNode{Ident: "S"},
			},
			ReturnTypes: []ast.TypeNode{intType},
			Body:        []ast.Node{},
		}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := h.HashNode(tt.node)
			if err != nil {
				t.Fatal(err)
			}
			if got == 0 {
				t.Fatal("zero hash")
			}
			got2, err := h.HashNode(tt.node)
			if err != nil {
				t.Fatal(err)
			}
			if got != got2 {
				t.Fatalf("not deterministic: %v vs %v", got, got2)
			}
		})
	}
}
