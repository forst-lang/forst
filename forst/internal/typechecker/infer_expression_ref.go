package typechecker

import (
	"fmt"
	"forst/internal/ast"
)

func (tc *TypeChecker) inferExpressionRefDeref(expr ast.Node) ([]ast.TypeNode, bool, error) {
	switch e := expr.(type) {
	case ast.ReferenceNode:

		valueType, err := tc.inferExpressionType(e.Value)
		if err != nil {
			return nil, true, err
		}
		referenceType := ast.TypeNode{
			Ident:      ast.TypePointer,
			TypeParams: valueType,
		}
		tc.storeInferredType(e, []ast.TypeNode{referenceType})
		return []ast.TypeNode{referenceType}, true, nil
	case ast.DereferenceNode:

		valueType, err := tc.inferExpressionType(e.Value)
		if err != nil {
			return nil, true, err
		}
		if len(valueType) != 1 {
			return nil, true, reportf(spanOfExpression(e.Value), "deref-type",
				"dereference requires a single pointer type",
				fmt.Sprintf("Dereference is only valid on a single type, got %s.", formatTypeList(valueType)),
				"use `*p` where `p` has exactly one pointer type")
		}
		tc.log.Tracef("Dereference type identifier: %+v", valueType[0].Node)
		if valueType[0].Ident != ast.TypePointer {
			return nil, true, reportf(spanOfExpression(e.Value), "deref-type",
				"dereference requires a pointer type",
				fmt.Sprintf("Cannot dereference type `%s`; expected a pointer.", valueType[0].Ident),
				"use `*p` on a pointer value")
		}
		tc.log.Tracef("Dereference type: %+v", valueType[0].TypeParams)
		tc.storeInferredType(e, valueType[0].TypeParams)
		return valueType[0].TypeParams, true, nil
	}
	return nil, false, nil
}
