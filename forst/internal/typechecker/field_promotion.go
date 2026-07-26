package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/ast"
)

func (tc *TypeChecker) shapePayloadForEmbeddedField(field ast.ShapeFieldNode) (*ast.ShapeNode, ast.TypeNode, error) {
	if field.IsMethod {
		return nil, ast.TypeNode{}, fmt.Errorf("embedded method fields are not supported")
	}
	if field.Shape != nil {
		return field.Shape, ast.TypeNode{Ident: ast.TypeShape}, nil
	}
	if field.Type == nil {
		return nil, ast.TypeNode{}, fmt.Errorf("embedded field has no type")
	}
	typ := tc.resolveTypeAliasChain(*field.Type)
	if payload, ok := tc.shapePayloadForType(typ); ok {
		return payload, typ, nil
	}
	return nil, typ, nil
}

func (tc *TypeChecker) shapePayloadForType(typ ast.TypeNode) (*ast.ShapeNode, bool) {
	def, ok := tc.Defs[typ.Ident]
	if !ok {
		if td, found := tc.typeDefForIdent(typ.Ident); found {
			def = td
			ok = true
		}
	}
	if !ok {
		return nil, false
	}
	typeDef, ok := def.(ast.TypeDefNode)
	if !ok {
		return nil, false
	}
	if payload, ok := ast.PayloadShape(typeDef.Expr); ok {
		return payload, true
	}
	return nil, false
}

func (tc *TypeChecker) lookupPromotedFieldInPayload(
	baseType ast.TypeNode,
	payload *ast.ShapeNode,
	fieldPath []string,
) (ast.TypeNode, error) {
	if payload == nil || len(fieldPath) == 0 {
		return ast.TypeNode{}, fmt.Errorf("invalid promoted field lookup")
	}
	var matches []string
	var result ast.TypeNode
	for _, embedName := range ast.ShapeFieldNamesInOrder(payload.Fields, payload.FieldOrder) {
		field := payload.Fields[embedName]
		if !field.Embedded {
			continue
		}
		innerShape, innerType, err := tc.shapePayloadForEmbeddedField(field)
		if err != nil {
			continue
		}
		var ft ast.TypeNode
		var ferr error
		if innerShape != nil {
			ft, ferr = tc.lookupFieldPathOnShape(innerShape, fieldPath)
		} else {
			ft, ferr = tc.lookupFieldPath(innerType, fieldPath)
		}
		if ferr != nil {
			continue
		}
		if len(matches) > 0 {
			return ast.TypeNode{}, fmt.Errorf("ambiguous selector %s in type %s", fieldPath[0], baseType.Ident)
		}
		matches = append(matches, embedName)
		result = ft
	}
	if len(matches) == 0 {
		return ast.TypeNode{}, fmt.Errorf("field path %v not found in type %s", fieldPath, baseType.Ident)
	}
	return result, nil
}

func (tc *TypeChecker) lookupPromotedFieldInShape(shape *ast.ShapeNode, fieldPath []string) (ast.TypeNode, error) {
	return tc.lookupPromotedFieldInPayload(ast.TypeNode{Ident: ast.TypeShape}, shape, fieldPath)
}

func isAmbiguousSelectorError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "ambiguous selector")
}
