package typechecker

import (
	"fmt"

	"forst/internal/ast"
)

func (tc *TypeChecker) inferConstGroup(n ast.ConstGroupNode) error {
	iotaIdx := 0
	var prevExpr ast.ExpressionNode
	for _, spec := range n.Specs {
		expr := spec.Value
		if expr == nil {
			expr = prevExpr
		} else {
			prevExpr = expr
		}
		if expr == nil {
			return fmt.Errorf("const %s: missing initializer", spec.Name.ID)
		}
		typ, err := tc.inferConstInit(expr, iotaIdx, spec.Type)
		if err != nil {
			return err
		}
		tc.registerPackageConst(spec.Name.ID, typ)
		iotaIdx++
	}
	return nil
}

func (tc *TypeChecker) inferConstInit(expr ast.ExpressionNode, iotaIdx int, explicit *ast.TypeNode) ([]ast.TypeNode, error) {
	substituted := substituteConstIota(expr, int64(iotaIdx))
	inferred, err := tc.inferExpressionType(substituted)
	if err != nil {
		return nil, fmt.Errorf("const initializer: %w", err)
	}
	if len(inferred) != 1 {
		return nil, fmt.Errorf("const initializer: expected a single type")
	}
	if explicit != nil && !tc.IsTypeCompatible(inferred[0], *explicit) {
		return nil, fmt.Errorf("const initializer type mismatch: got %s, expected %s", inferred[0].Ident, explicit.Ident)
	}
	if explicit != nil {
		return []ast.TypeNode{*explicit}, nil
	}
	return inferred, nil
}

func substituteConstIota(expr ast.ExpressionNode, iotaVal int64) ast.ExpressionNode {
	switch e := expr.(type) {
	case ast.IotaLiteralNode:
		return ast.IntLiteralNode{Value: iotaVal}
	case ast.BinaryExpressionNode:
		return ast.BinaryExpressionNode{
			Left:     substituteConstIota(e.Left, iotaVal),
			Operator: e.Operator,
			Right:    substituteConstIota(e.Right, iotaVal),
		}
	case ast.UnaryExpressionNode:
		return ast.UnaryExpressionNode{
			Operator: e.Operator,
			Operand:  substituteConstIota(e.Operand, iotaVal),
		}
	default:
		return expr
	}
}
