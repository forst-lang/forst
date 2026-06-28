package typechecker

import (
	"forst/internal/ast"
)

// ShapeFieldsFromParamType returns shape fields for a destructured parameter type
// (inline `{ ... }`, typedef, or assertion chain such as AppMutation.Input({ ... })).
func (tc *TypeChecker) ShapeFieldsFromParamType(typeNode ast.TypeNode) (map[string]ast.ShapeFieldNode, bool) {
	if typeNode.Ident == ast.TypeShape && typeNode.Assertion != nil {
		for _, c := range typeNode.Assertion.Constraints {
			if c.Name == ConstraintMatch && len(c.Args) > 0 && c.Args[0].Shape != nil {
				return c.Args[0].Shape.Fields, true
			}
			if c.Name == "Shape" && len(c.Args) > 0 && c.Args[0].Shape != nil {
				return c.Args[0].Shape.Fields, true
			}
		}
	}

	if typeNode.Assertion != nil {
		fields := tc.resolveShapeFieldsFromAssertion(typeNode.Assertion)
		if len(fields) > 0 {
			return fields, true
		}
	}

	if def, ok := tc.Defs[typeNode.Ident]; ok {
		if shape, ok := tc.getShapeFromTypeDef(def); ok {
			return shape.Fields, true
		}
		if typeDef, ok := def.(ast.TypeDefNode); ok {
			if ae, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok && ae.Assertion != nil {
				fields := tc.resolveShapeFieldsFromAssertion(ae.Assertion)
				if len(fields) > 0 {
					return fields, true
				}
			}
		}
	}

	return nil, false
}

// ShapeFieldTypeNode extracts a TypeNode from a shape field definition.
func ShapeFieldTypeNode(field ast.ShapeFieldNode) (ast.TypeNode, bool) {
	if field.Type != nil {
		return *field.Type, true
	}
	if field.Assertion != nil {
		return ast.TypeNode{
			Ident:     ast.TypeAssertion,
			Assertion: field.Assertion,
		}, true
	}
	if field.Shape != nil {
		baseType := ast.TypeIdent(ast.TypeShape)
		return ast.TypeNode{
			Ident: ast.TypeShape,
			Assertion: &ast.AssertionNode{
				BaseType: &baseType,
				Constraints: []ast.ConstraintNode{{
					Name: ConstraintMatch,
					Args: []ast.ConstraintArgumentNode{{
						Shape: field.Shape,
					}},
				}},
			},
		}, true
	}
	return ast.TypeNode{}, false
}

func (tc *TypeChecker) registerDestructuredParamSymbols(fields []string, paramType ast.TypeNode, symbolKind SymbolKind) {
	shapeFields, ok := tc.ShapeFieldsFromParamType(paramType)
	if !ok {
		return
	}
	for _, name := range fields {
		sf, ok := shapeFields[name]
		if !ok {
			continue
		}
		if tn, ok := ShapeFieldTypeNode(sf); ok {
			tc.storeSymbol(ast.Identifier(name), []ast.TypeNode{tn}, symbolKind)
		}
	}
}
