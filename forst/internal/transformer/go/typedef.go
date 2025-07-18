package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"

	logrus "github.com/sirupsen/logrus"
)

func (t *Transformer) transformTypeDef(node ast.TypeDefNode) (*goast.GenDecl, error) {
	hash, err := t.TypeChecker.Hasher.HashNode(node)
	if err != nil {
		return nil, fmt.Errorf("failed to hash type node: %w", err)
	}
	hashTypeName := hash.ToTypeIdent()

	expr, err := t.transformTypeDefExpr(node.Expr)
	if err != nil {
		t.log.WithFields(logrus.Fields{
			"function": "transformTypeDef",
		}).WithError(err).Error("failed to transform type def expr during transformation")
		return nil, err
	}

	// Use original name for user-defined types, hash-based name for structural types
	var typeName ast.TypeIdent
	var typeNode = ast.TypeNode{Ident: node.Ident}
	if typeNode.IsHashBased() {
		// For hash-based types, use the hash-based name
		typeName = hashTypeName
	} else {
		// For user-defined types, use the original name
		typeName = node.Ident
	}

	// For comments, always use the original Forst type name
	commentName := string(node.Ident)

	comments := []*goast.Comment{
		{
			Text: fmt.Sprintf("// %s: %s", commentName, node.Expr.String()),
		},
	}

	return &goast.GenDecl{
		Tok: token.TYPE,
		Specs: []goast.Spec{
			&goast.TypeSpec{
				Name: &goast.Ident{
					Name: string(typeName),
				},
				Type: *expr,
			},
		},
		Doc: &goast.CommentGroup{
			List: comments,
		},
	}, nil
}

func (t *Transformer) getAssertionBaseTypeIdent(assertion *ast.AssertionNode) (*goast.Ident, error) {
	if assertion.BaseType != nil {
		ident, err := transformTypeIdent(*assertion.BaseType)
		if err != nil {
			err = fmt.Errorf("failed to transform type ident during getAssertionBaseTypeIdent: %w", err)
			t.log.WithFields(logrus.Fields{
				"function": "getAssertionBaseTypeIdent",
			}).WithError(err).Error("transforming assertion base type ident failed")
			return nil, err
		}
		return ident, nil
	}

	typeNode, err := t.TypeChecker.LookupAssertionType(assertion)
	if err != nil {
		err = fmt.Errorf("failed to lookup assertion type during getAssertionBaseTypeIdent: %w", err)
		t.log.WithFields(logrus.Fields{
			"function": "getAssertionBaseTypeIdent",
		}).WithError(err).Error("transforming assertion base type ident failed")
		return nil, err
	}

	ident, err := transformTypeIdent(typeNode.Ident)
	if err != nil {
		err = fmt.Errorf("failed to transform type ident during getAssertionBaseTypeIdent: %w", err)
		t.log.WithFields(logrus.Fields{
			"function": "getAssertionBaseTypeIdent",
		}).WithError(err).Error("transforming assertion base type ident failed")
		return nil, err
	}
	return ident, nil
}

// TODO: Improve shape type registration
// This should handle nested shapes better and generate appropriate validation code
func (t *Transformer) defineShapeType(shape *ast.ShapeNode) error {
	// First register all nested shape fields
	if err := t.defineShapeFields(shape); err != nil {
		return err
	}

	// Then register the shape itself
	hash, err := t.TypeChecker.Hasher.HashNode(shape)
	if err != nil {
		return fmt.Errorf("failed to hash shape during defineShapeType: %w", err)
	}
	typeIdent := hash.ToTypeIdent()

	// Create struct type for the shape
	structType, err := t.transformShapeType(shape)
	if err != nil {
		return fmt.Errorf("failed to transform shape type %s: %w", typeIdent, err)
	}

	// Use the unified aliasing logic for the type name
	aliasName, err := t.TypeChecker.GetAliasedTypeName(ast.TypeNode{Ident: typeIdent})
	if err != nil || aliasName == "" {
		aliasName = string(typeIdent)
	}

	decl, err := t.transformTypeDef(ast.TypeDefNode{
		Ident: ast.TypeIdent(aliasName),
		Expr: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: nil,
				Constraints: []ast.ConstraintNode{{
					Name: "Shape",
					Args: []ast.ConstraintArgumentNode{{
						Shape: shape,
					}},
				}},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to transform shape type %s: %w", typeIdent, err)
	}

	// Override the type with our struct type
	decl.Specs[0].(*goast.TypeSpec).Type = *structType
	t.Output.AddType(decl)
	return nil
}

// defineShapeFields recursively registers type definitions for all shape fields
func (t *Transformer) defineShapeFields(shape *ast.ShapeNode) error {
	for _, field := range shape.Fields {
		if field.Shape != nil {
			// Recursively register nested shapes
			if err := t.defineShapeFields(field.Shape); err != nil {
				return err
			}

			// Register the shape field type
			hash, err := t.TypeChecker.Hasher.HashNode(field.Shape)
			if err != nil {
				return fmt.Errorf("failed to hash shape field type %s: %w", field.Shape.String(), err)
			}
			typeIdent := hash.ToTypeIdent()

			// Create struct type for the shape field
			structType, err := t.transformShapeType(field.Shape)
			if err != nil {
				return fmt.Errorf("failed to transform shape field type %s: %w", typeIdent, err)
			}

			// Use the unified aliasing logic for the type name
			aliasName, err := t.TypeChecker.GetAliasedTypeName(ast.TypeNode{Ident: typeIdent})
			if err != nil || aliasName == "" {
				aliasName = string(typeIdent)
			}

			decl, err := t.transformTypeDef(ast.TypeDefNode{
				Ident: ast.TypeIdent(aliasName),
				Expr: ast.TypeDefAssertionExpr{
					Assertion: &ast.AssertionNode{
						BaseType: nil,
						Constraints: []ast.ConstraintNode{{
							Name: "Shape",
							Args: []ast.ConstraintArgumentNode{{
								Shape: field.Shape,
							}},
						}},
					},
				},
			})
			if err != nil {
				return fmt.Errorf("failed to transform shape field type %s: %w", typeIdent, err)
			}

			// Override the type with our struct type
			decl.Specs[0].(*goast.TypeSpec).Type = *structType
			t.Output.AddType(decl)
		}
	}
	return nil
}

// defineShapeTypes finds all shapes in type definitions and registers them
func (t *Transformer) defineShapeTypes() error {
	for _, def := range t.TypeChecker.Defs {
		if typeDef, ok := def.(ast.TypeDefNode); ok {
			if assertionExpr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
				if assertionExpr.Assertion != nil {
					for _, constraint := range assertionExpr.Assertion.Constraints {
						if len(constraint.Args) > 0 && constraint.Args[0].Shape != nil {
							if err := t.defineShapeType(constraint.Args[0].Shape); err != nil {
								return err
							}
						}
					}
				}
			}
		}
	}
	return nil
}

// getTypeAliasNameForTypeNode uses the unified type aliasing function from the typechecker
func (t *Transformer) getTypeAliasNameForTypeNode(typeNode ast.TypeNode) (string, error) {
	// Handle Shape and TYPE_SHAPE types specially - emit inline empty struct
	if typeNode.Ident == ast.TypeIdent("Shape") || typeNode.Ident == ast.TypeIdent("TYPE_SHAPE") {
		t.log.Debugf("getTypeAliasNameForTypeNode: Shape/TYPE_SHAPE type %q -> inline struct", typeNode.Ident)
		return "struct{}", nil
	}

	// Use the unified type aliasing function from the typechecker
	return t.TypeChecker.GetAliasedTypeName(typeNode)
}

// getGeneratedTypeNameForTypeNode uses the unified type aliasing function from the typechecker
func (t *Transformer) getGeneratedTypeNameForTypeNode(typeNode ast.TypeNode) (string, error) {
	// Use the unified type aliasing function from the typechecker
	return t.TypeChecker.GetAliasedTypeName(typeNode)
}

// getAliasedTypeNameForTypeNode returns the aliased type name for any type node,
// ensuring consistent type aliasing across the transformer
func (t *Transformer) getAliasedTypeNameForTypeNode(typeNode ast.TypeNode) (string, error) {
	// Use the unified type aliasing function from the typechecker
	return t.TypeChecker.GetAliasedTypeName(typeNode)
}
