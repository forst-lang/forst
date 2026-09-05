package typechecker

import (
	"fmt"
	"forst/internal/ast"
)

func (tc *TypeChecker) inferExpressionArrayLiteral(expr ast.Node) ([]ast.TypeNode, bool, error) {
	e, ok := expr.(ast.ArrayLiteralNode)
	if !ok {
		return nil, false, nil
	}
	if len(e.Value) == 0 {
		elem := ast.TypeNode{Ident: ast.TypeInt}
		if e.Type.Ident != ast.TypeImplicit && e.Type.Ident != "" {
			elem = e.Type
		}
		arr := ast.TypeNode{Ident: ast.TypeArray, TypeParams: []ast.TypeNode{elem}}
		tc.storeInferredType(e, []ast.TypeNode{arr})
		return []ast.TypeNode{arr}, true, nil
	}
	var elemType ast.TypeNode
	for i, el := range e.Value {
		ts, err := tc.inferExpressionType(el)
		if err != nil {
			return nil, true, err
		}
		if len(ts) != 1 {
			elSpan := spanOfExpression(el)
			return nil, true, reportf(elSpan, "array-element-type",
				fmt.Sprintf("array element %d must have a single type", i),
				fmt.Sprintf("Element %d of the array literal must infer to exactly one type.", i),
				"ensure each element has one type")
		}
		if i == 0 {
			elemType = ts[0]
		} else if elemType.Ident != ts[0].Ident {
			elSpan := spanOfExpression(el)
			return nil, true, reportf(elSpan, "array-mixed-types",
				"array literal has mixed element types",
				fmt.Sprintf("Array elements must share one type; got `%s` and `%s`.", formatTypeIdentForDiag(elemType.Ident), formatTypeIdentForDiag(ts[0].Ident)),
				"use the same type for every element or add an explicit element type")
		}
	}
	if e.Type.Ident != ast.TypeImplicit && e.Type.Ident != "" {
		elemType = e.Type
	}
	arr := ast.TypeNode{Ident: ast.TypeArray, TypeParams: []ast.TypeNode{elemType}}
	tc.storeInferredType(e, []ast.TypeNode{arr})
	return []ast.TypeNode{arr}, true, nil
}

func (tc *TypeChecker) inferExpressionMapLiteral(expr ast.Node) ([]ast.TypeNode, bool, error) {
	e, ok := expr.(ast.MapLiteralNode)
	if !ok {
		return nil, false, nil
	}
	mapSpan := ast.SourceSpan{}
	if len(e.Entries) > 0 {
		mapSpan = spanOfExpression(e.Entries[0].Key)
	}
	if e.Type.Ident != ast.TypeMap || len(e.Type.TypeParams) != 2 {
		return nil, true, reportf(mapSpan, "map-literal-type",
			"map literal has invalid type",
			fmt.Sprintf("Map literal type must be Map(K, V); got %v.", e.Type),
			"annotate the literal as Map(keyType, valueType)")
	}
	wantK, wantV := e.Type.TypeParams[0], e.Type.TypeParams[1]
	for i, ent := range e.Entries {
		entrySpan := firstSetSpan(spanOfExpression(ent.Key), spanOfExpression(ent.Value), mapSpan)
		kt, err := tc.inferExpressionType(ent.Key)
		if err != nil {
			return nil, true, err
		}
		if len(kt) != 1 {
			return nil, true, reportf(entrySpan, "map-literal-key-type",
				fmt.Sprintf("map literal entry %d key must have one type", i),
				fmt.Sprintf("Key %d must infer to exactly one type.", i),
				"ensure each map key has a single type")
		}
		if !tc.IsTypeCompatible(kt[0], wantK) {
			return nil, true, reportf(entrySpan, "map-literal-key-type",
				fmt.Sprintf("map literal entry %d key type mismatch", i),
				fmt.Sprintf("Key %d must have type `%s`, got `%s`.", i, formatTypeIdentForDiag(wantK.Ident), formatTypeIdentForDiag(kt[0].Ident)),
				"convert the key or change the map key type")
		}
		vt, err := tc.inferExpressionType(ent.Value)
		if err != nil {
			return nil, true, err
		}
		if len(vt) != 1 {
			return nil, true, reportf(entrySpan, "map-literal-value-type",
				fmt.Sprintf("map literal entry %d value must have one type", i),
				fmt.Sprintf("Value %d must infer to exactly one type.", i),
				"ensure each map value has a single type")
		}
		if !tc.IsTypeCompatible(vt[0], wantV) {
			return nil, true, reportf(entrySpan, "map-literal-value-type",
				fmt.Sprintf("map literal entry %d value type mismatch", i),
				fmt.Sprintf("Value %d must have type `%s`, got `%s`.", i, formatTypeIdentForDiag(wantV.Ident), formatTypeIdentForDiag(vt[0].Ident)),
				"convert the value or change the map value type")
		}
	}
	tc.storeInferredType(e, []ast.TypeNode{e.Type})
	return []ast.TypeNode{e.Type}, true, nil
}
