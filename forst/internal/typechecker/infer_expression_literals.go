package typechecker

import (
	"forst/internal/ast"
)

func (tc *TypeChecker) inferExpressionLiterals(expr ast.Node) ([]ast.TypeNode, bool, error) {
	switch e := expr.(type) {
	case ast.IntLiteralNode:

		typ := ast.TypeNode{Ident: ast.TypeInt}
		tc.storeInferredType(e, []ast.TypeNode{typ})
		return []ast.TypeNode{typ}, true, nil
	case ast.FloatLiteralNode:

		typ := ast.TypeNode{Ident: ast.TypeFloat}
		tc.storeInferredType(e, []ast.TypeNode{typ})
		return []ast.TypeNode{typ}, true, nil
	case ast.StringLiteralNode:

		typ := ast.TypeNode{Ident: ast.TypeString}
		tc.storeInferredType(e, []ast.TypeNode{typ})
		return []ast.TypeNode{typ}, true, nil
	case ast.RuneLiteralNode:

		typ := ast.TypeNode{Ident: ast.TypeInt}
		tc.storeInferredType(e, []ast.TypeNode{typ})
		return []ast.TypeNode{typ}, true, nil
	case ast.BoolLiteralNode:

		typ := ast.TypeNode{Ident: ast.TypeBool}
		tc.storeInferredType(e, []ast.TypeNode{typ})
		return []ast.TypeNode{typ}, true, nil
	case ast.NilLiteralNode:

		// Return a special marker (empty slice) to indicate untyped nil; context must resolve
		return nil, true, nil
	case ast.IotaLiteralNode:

		return nil, true, reportf(spanOfNode(e), "iota-outside-const",
			"iota is only valid in const declarations",
			"The predeclared `iota` may only appear in a const group.",
			"move `iota` into a const declaration")
	}
	return nil, false, nil
}
