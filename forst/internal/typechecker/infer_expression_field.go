package typechecker

import (
	"forst/internal/ast"
	"go/types"
	"strconv"
)

func (tc *TypeChecker) inferExpressionFieldAccess(expr ast.Node) ([]ast.TypeNode, bool, error) {
	switch e := expr.(type) {
	case ast.FieldAccessNode:

		targetTypes, err := tc.inferExpressionType(e.Target)
		if err != nil {
			return nil, true, err
		}
		if len(targetTypes) == 1 && targetTypes[0].IsTupleType() {
			idx, convErr := strconv.Atoi(string(e.Field.ID))
			if convErr != nil || idx < 0 || idx >= len(targetTypes[0].TypeParams) {
				return nil, true, reportBodyf(e.Field.Span, "tuple-index", "tuple index %s out of range for %s", e.Field.ID, targetTypes[0].String())
			}
			ft := targetTypes[0].TypeParams[idx]
			tc.storeInferredType(e, []ast.TypeNode{ft})
			return []ast.TypeNode{ft}, true, nil
		}
		if goRecv := tc.goTypeForExpression(e.Target); goRecv != nil {
			obj, _, _ := types.LookupFieldOrMethod(goRecv, false, nil, string(e.Field.ID))
			if obj == nil {
				return nil, true, reportBodyf(e.Field.Span, "go-field", "%s has no field %s", goRecv.String(), e.Field.ID)
			}
			ft, ok := tc.mapGoType(obj.Type())
			if !ok {
				return nil, true, reportBodyf(e.Field.Span, "go-field", "cannot map Go field type %s", obj.Type().String())
			}
			tc.storeInferredType(e, []ast.TypeNode{ft})
			return []ast.TypeNode{ft}, true, nil
		}
		if len(targetTypes) != 1 {
			return nil, true, reportBodyf(e.Field.Span, "field-access", "field access requires a single target type")
		}
		ft, err := tc.lookupFieldPath(targetTypes[0], []string{string(e.Field.ID)}, e.Field.Span)
		if err != nil {
			return nil, true, err
		}
		tc.storeInferredType(e, []ast.TypeNode{ft})
		return []ast.TypeNode{ft}, true, nil
	}
	return nil, false, nil
}
