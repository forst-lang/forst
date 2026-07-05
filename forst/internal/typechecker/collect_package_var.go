package typechecker

import (
	"fmt"

	"forst/internal/ast"
)

// literalExpressionTypes returns types for simple literal expressions without full inference.
func literalExpressionTypes(expr ast.ExpressionNode) ([]ast.TypeNode, bool) {
	switch e := expr.(type) {
	case ast.IntLiteralNode:
		_ = e
		return []ast.TypeNode{{Ident: ast.TypeInt}}, true
	case ast.FloatLiteralNode:
		return []ast.TypeNode{{Ident: ast.TypeFloat}}, true
	case ast.StringLiteralNode:
		return []ast.TypeNode{{Ident: ast.TypeString}}, true
	case ast.BoolLiteralNode:
		return []ast.TypeNode{{Ident: ast.TypeBool}}, true
	case ast.NilLiteralNode:
		return nil, true
	default:
		return nil, false
	}
}

func (tc *TypeChecker) collectPackageLevelVar(n ast.AssignmentNode) error {
	if len(n.LValues) != 1 {
		return fmt.Errorf("package-level var: expected one name")
	}
	vn, ok := n.LValues[0].(ast.VariableNode)
	if !ok {
		return fmt.Errorf("package-level var: name must be a simple identifier")
	}
	var typ []ast.TypeNode
	if len(n.ExplicitTypes) > 0 && n.ExplicitTypes[0] != nil {
		typ = []ast.TypeNode{*n.ExplicitTypes[0]}
	} else if vn.ExplicitType.Ident != "" {
		typ = []ast.TypeNode{vn.ExplicitType}
	} else if len(n.RValues) > 0 {
		if ts, ok := literalExpressionTypes(n.RValues[0]); ok {
			typ = ts
		}
		// Non-literal initializers are registered during InferTypes.
	}
	if len(typ) == 0 {
		return nil
	}
	tc.storeSymbol(vn.Ident.ID, typ, SymbolVariable)
	return nil
}

func (tc *TypeChecker) ensurePackageLevelVarRegistered(n ast.AssignmentNode) error {
	if len(n.LValues) != 1 {
		return fmt.Errorf("package-level var: expected one name")
	}
	vn, ok := n.LValues[0].(ast.VariableNode)
	if !ok {
		return fmt.Errorf("package-level var: name must be a simple identifier")
	}
	if _, exists := tc.globalScope().LookupVariableType(vn.Ident.ID); exists {
		return nil
	}
	var typ []ast.TypeNode
	if len(n.ExplicitTypes) > 0 && n.ExplicitTypes[0] != nil {
		typ = []ast.TypeNode{*n.ExplicitTypes[0]}
	} else if vn.ExplicitType.Ident != "" {
		typ = []ast.TypeNode{vn.ExplicitType}
	} else if len(n.RValues) > 0 {
		ts, err := tc.inferExpressionType(n.RValues[0])
		if err != nil {
			return err
		}
		typ = ts
	}
	if len(typ) == 0 {
		return fmt.Errorf("package-level var %s: could not infer type", vn.Ident.ID)
	}
	tc.storeSymbol(vn.Ident.ID, typ, SymbolVariable)
	return nil
}
