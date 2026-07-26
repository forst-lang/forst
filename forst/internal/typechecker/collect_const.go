package typechecker

import (
	"forst/internal/ast"
)

func (tc *TypeChecker) collectConstGroup(n ast.ConstGroupNode) error {
	for _, spec := range n.Specs {
		if spec.Value == nil || containsIota(spec.Value) {
			continue
		}
		var typ []ast.TypeNode
		if spec.Type != nil {
			typ = []ast.TypeNode{*spec.Type}
		} else if ts, ok := literalExpressionTypes(spec.Value); ok && len(ts) > 0 {
			typ = ts
		}
		if len(typ) == 0 {
			continue
		}
		tc.registerPackageConst(spec.Name.ID, typ)
	}
	return nil
}

func (tc *TypeChecker) registerPackageConst(name ast.Identifier, typ []ast.TypeNode) {
	if tc.packageConsts == nil {
		tc.packageConsts = make(map[ast.Identifier]struct{})
	}
	tc.packageConsts[name] = struct{}{}
	tc.storeSymbol(name, typ, SymbolVariable)
	tc.VariableTypes[name] = append([]ast.TypeNode(nil), typ...)
}

func (tc *TypeChecker) isPackageConst(name ast.Identifier) bool {
	if tc.packageConsts == nil {
		return false
	}
	_, ok := tc.packageConsts[name]
	return ok
}

func containsIota(expr ast.ExpressionNode) bool {
	switch e := expr.(type) {
	case ast.IotaLiteralNode:
		return true
	case ast.BinaryExpressionNode:
		return containsIota(e.Left) || containsIota(e.Right)
	case ast.UnaryExpressionNode:
		return containsIota(e.Operand)
	default:
		return false
	}
}
