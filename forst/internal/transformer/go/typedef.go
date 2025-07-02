package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"

	logrus "github.com/sirupsen/logrus"
)

func (t *Transformer) transformTypeDef(node ast.TypeDefNode) (*goast.GenDecl, error) {
	expr, err := t.transformTypeDefExpr(node.Expr)
	if err != nil {
		t.log.WithFields(logrus.Fields{
			"function": "transformTypeDef",
		}).WithError(err).Error("failed to transform type def expr during transformation")
		return nil, err
	}

	// Simple rule: use the original identifier name for all types
	typeName := string(node.Ident)

	// Fix: Avoid recursive type aliasing for assertion types
	if aliasIdent, ok := (*expr).(*goast.Ident); ok && aliasIdent.Name == typeName {
		if assertionExpr, ok := node.Expr.(ast.TypeDefAssertionExpr); ok && assertionExpr.Assertion != nil {
			if assertionExpr.Assertion.BaseType != nil {
				underlying := string(*assertionExpr.Assertion.BaseType)
				if underlying != typeName {
					aliasIdent.Name = underlying
				}
			}
		}
	}

	return &goast.GenDecl{
		Tok: token.TYPE,
		Specs: []goast.Spec{
			&goast.TypeSpec{
				Name: &goast.Ident{
					Name: typeName,
				},
				Type: *expr,
			},
		},
		Doc: &goast.CommentGroup{
			List: []*goast.Comment{
				{
					Text: fmt.Sprintf("// %s: %s", typeName, node.Expr.String()),
				},
			},
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

	decl, err := t.transformTypeDef(ast.TypeDefNode{
		Ident: typeIdent,
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

			decl, err := t.transformTypeDef(ast.TypeDefNode{
				Ident: typeIdent,
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

// getTypeAliasNameForTypeNode returns the type name for a given TypeNode.
// Simple rule: use hash-based names consistently for all types except built-ins.
func (t *Transformer) getTypeAliasNameForTypeNode(typeNode ast.TypeNode) (string, error) {
	// If the type is a built-in, return its Go name
	if isGoBuiltinType(string(typeNode.Ident)) || typeNode.Ident == ast.TypeString || typeNode.Ident == ast.TypeInt || typeNode.Ident == ast.TypeFloat || typeNode.Ident == ast.TypeBool || typeNode.Ident == ast.TypeVoid || typeNode.Ident == ast.TypeError {
		ident, err := transformTypeIdent(typeNode.Ident)
		if err != nil {
			err = fmt.Errorf("failed to transform type ident during getTypeAliasNameForTypeNode: %w", err)
			t.log.WithFields(logrus.Fields{
				"function": "getTypeAliasNameForTypeNode",
			}).WithError(err).Error("transforming type ident failed")
			return "", err
		}
		if ident != nil {
			return ident.Name, nil
		}
		return string(typeNode.Ident), nil
	}

	// Handle pointer types
	if typeNode.Ident == ast.TypePointer {
		if len(typeNode.TypeParams) == 0 {
			return "", fmt.Errorf("pointer type must have a base type parameter")
		}
		baseTypeName, err := t.getTypeAliasNameForTypeNode(typeNode.TypeParams[0])
		if err != nil {
			return "", fmt.Errorf("failed to get base type name for pointer: %w", err)
		}
		return "*" + baseTypeName, nil
	}

	// For all other types, use the original identifier name
	// This includes both explicitly named types (like AppContext) and hash-based types (like T_488eVThFocF)
	return string(typeNode.Ident), nil
}
