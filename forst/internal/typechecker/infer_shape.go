package typechecker

import (
	"fmt"
	"forst/internal/ast"

	log "github.com/sirupsen/logrus"
)

// TODO: Improve type inference for complex types
// This should handle:
// 1. Binary type expressions
// 2. Nested shapes
// 3. Type aliases
// 4. Generic types
func (tc *TypeChecker) inferShapeType(shape *ast.ShapeNode) ([]ast.TypeNode, error) {
	hash := tc.Hasher.HashNode(shape)
	typeIdent := hash.ToTypeIdent()
	shapeType := []ast.TypeNode{
		{
			Ident: typeIdent,
		},
	}
	for name, field := range shape.Fields {
		if field.Shape != nil {
			fieldType, err := tc.inferShapeType(field.Shape)
			if err != nil {
				return nil, err
			}

			fieldHash := tc.Hasher.HashNode(field)
			fieldTypeIdent := fieldHash.ToTypeIdent()
			log.Tracef("Inferred type of shape field %s: %s, field: %s", name, fieldTypeIdent, field)
			tc.storeInferredType(field.Shape, fieldType)
		} else if field.Assertion != nil {
			// Skip if the assertion type has already been inferred
			inferredType, _ := tc.inferAssertionType(field.Assertion, false)
			if inferredType != nil {
				continue
			}

			fieldHash := tc.Hasher.HashNode(field)
			fieldTypeIdent := fieldHash.ToTypeIdent()
			log.Tracef("Inferred type of assertion field %s: %s", name, fieldTypeIdent)
			tc.registerType(ast.TypeDefNode{
				Ident: fieldTypeIdent,
				Expr: ast.TypeDefAssertionExpr{
					Assertion: field.Assertion,
				},
			})
		} else {
			panic(fmt.Sprintf("Shape field has neither assertion nor shape: %T", field))
		}
	}

	tc.storeInferredType(shape, shapeType)
	log.Tracef("Inferred shape type: %s", shapeType)

	// The type is not registered here as the specific type implementation
	// depends on the target language and will be determined in the transformer.

	return shapeType, nil
}
