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
	if ast.IsHashBasedType(ast.TypeNode{Ident: node.Ident}) {
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

// This rule is enforced below:
func (t *Transformer) getTypeAliasNameForTypeNode(typeNode ast.TypeNode) (string, error) {
	t.log.Debugf("getTypeAliasNameForTypeNode: input type %q, typeKind %v", typeNode.Ident, typeNode.TypeKind)

	// Handle Shape and TYPE_SHAPE types specially - emit inline empty struct
	if typeNode.Ident == ast.TypeIdent("Shape") || typeNode.Ident == ast.TypeIdent("TYPE_SHAPE") {
		t.log.Debugf("getTypeAliasNameForTypeNode: Shape/TYPE_SHAPE type %q -> inline struct", typeNode.Ident)
		return "struct{}", nil
	}

	// Ensure the type is registered before emitting
	if typeNode.TypeKind == ast.TypeKindHashBased || typeNode.TypeKind == ast.TypeKindUserDefined {
		if _, exists := t.TypeChecker.Defs[typeNode.Ident]; !exists {
			t.log.Warnf("Type %q not registered in Defs; synthesizing minimal definition", typeNode.Ident)
			// Synthesize a minimal struct type
			minimal := ast.TypeDefNode{
				Ident: typeNode.Ident,
				Expr:  ast.TypeDefShapeExpr{Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}},
			}
			t.TypeChecker.RegisterTypeIfMissing(typeNode.Ident, minimal)
		}
	}

	switch typeNode.TypeKind {
	case ast.TypeKindBuiltin:
		// Handle pointer types specially - they should be processed by transformType, not here
		if typeNode.Ident == ast.TypePointer {
			panic("BUG: pointer types should be handled by transformType, not getTypeAliasNameForTypeNode")
		}
		ident, err := transformTypeIdent(typeNode.Ident)
		if err != nil {
			return "", fmt.Errorf("failed to transform type ident during getTypeAliasNameForTypeNode: %w", err)
		}
		if ident != nil {
			result := ident.Name
			t.log.Debugf("getTypeAliasNameForTypeNode: builtin type %q -> alias %q", typeNode.Ident, result)
			return result, nil
		}
		return string(typeNode.Ident), nil
	case ast.TypeKindHashBased:
		return string(typeNode.Ident), nil
	case ast.TypeKindUserDefined:
		// Always emit the hash-based name for user-defined types
		hash, err := t.TypeChecker.Hasher.HashNode(typeNode)
		if err != nil {
			return "", fmt.Errorf("failed to hash type node: %w", err)
		}
		result := string(hash.ToTypeIdent())
		t.log.Debugf("getTypeAliasNameForTypeNode: user-defined type %q -> hash-based alias %q", typeNode.Ident, result)
		return result, nil
	default:
		return string(typeNode.Ident), nil
	}
}

// getGeneratedTypeNameForTypeNode looks up the generated type name for a TypeNode
// by checking if it's already defined in the typechecker's Defs
func (t *Transformer) getGeneratedTypeNameForTypeNode(typeNode ast.TypeNode) (string, error) {
	// For built-in types, return their Go names
	if isGoBuiltinType(string(typeNode.Ident)) || typeNode.Ident == ast.TypeString || typeNode.Ident == ast.TypeInt || typeNode.Ident == ast.TypeFloat || typeNode.Ident == ast.TypeBool || typeNode.Ident == ast.TypeVoid || typeNode.Ident == ast.TypeError {
		ident, err := transformTypeIdent(typeNode.Ident)
		if err != nil {
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
		baseTypeName, err := t.getGeneratedTypeNameForTypeNode(typeNode.TypeParams[0])
		if err != nil {
			return "", fmt.Errorf("failed to get base type name for pointer: %w", err)
		}
		return "*" + baseTypeName, nil
	}

	// For user-defined types, check if they're already defined in the typechecker
	if typeNode.Ident != "" {
		// Check if this type is already defined in the typechecker's Defs
		if _, exists := t.TypeChecker.Defs[typeNode.Ident]; exists {
			// Return the type name as-is since it's already defined
			return string(typeNode.Ident), nil
		}
	}

	// If not found in Defs, fall back to hash-based name
	hash, err := t.TypeChecker.Hasher.HashNode(typeNode)
	if err != nil {
		return "", fmt.Errorf("failed to hash type node: %w", err)
	}
	return string(hash.ToTypeIdent()), nil
}
