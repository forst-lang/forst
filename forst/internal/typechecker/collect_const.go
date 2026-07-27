package typechecker

import (
	"forst/internal/ast"
)

func (tc *TypeChecker) collectConstGroup(n ast.ConstGroupNode) error {
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
			iotaIdx++
			continue
		}
		var typ []ast.TypeNode
		if spec.Type != nil {
			typ = []ast.TypeNode{*spec.Type}
		} else if ts, err := tc.inferConstInit(expr, iotaIdx, nil); err == nil && len(ts) > 0 {
			typ = ts
		} else if ts, ok := literalExpressionTypes(expr); ok && len(ts) > 0 {
			typ = ts
		} else if containsIota(expr) {
			// Iota-backed consts fold to Int when no explicit type is given.
			typ = []ast.TypeNode{ast.TypeNode{Ident: ast.TypeInt}}
		}
		if len(typ) > 0 {
			tc.registerPackageConst(spec.Name.ID, typ)
		}
		iotaIdx++
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

// IsTopLevelPackageConst reports whether id is a package-level const.
func (tc *TypeChecker) IsTopLevelPackageConst(id ast.Identifier) bool {
	return tc.isPackageConst(id)
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
