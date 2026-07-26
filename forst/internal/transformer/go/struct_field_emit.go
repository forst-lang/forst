package transformergo

import "forst/internal/ast"

func (t *Transformer) goShapeDataFieldName(forstName string, field ast.ShapeFieldNode) string {
	if field.Embedded {
		return forstName
	}
	if t.ExportReturnStructFields || field.GoExport {
		return capitalizeFirst(forstName)
	}
	return forstName
}

func (t *Transformer) goSelectorFieldName(ownerTypes []ast.TypeNode, forstField string) string {
	if t.ExportReturnStructFields {
		return capitalizeFirst(forstField)
	}
	if len(ownerTypes) == 1 {
		if field, ok := t.TypeChecker.ShapeFieldForNamedShape(ownerTypes[0].Ident, forstField); ok {
			return t.goShapeDataFieldName(forstField, field)
		}
	}
	return forstField
}

func (t *Transformer) ownerTypesAfterField(ownerTypes []ast.TypeNode, forstField string) []ast.TypeNode {
	if len(ownerTypes) != 1 {
		return nil
	}
	ft, ok := t.TypeChecker.FieldTypeForNamedShape(ownerTypes[0].Ident, forstField)
	if !ok {
		return nil
	}
	return []ast.TypeNode{ft}
}
