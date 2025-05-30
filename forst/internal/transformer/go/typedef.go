package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"

	log "github.com/sirupsen/logrus"
)

func (t *Transformer) transformTypeDef(node ast.TypeDefNode) (*goast.GenDecl, error) {
	expr, err := t.transformTypeDefExpr(node.Expr)
	if err != nil {
		log.Error(fmt.Errorf("failed to transform type def expr during transformation: %s", err))
		return nil, err
	}

	// Always use hash-based names for types, but hash only the expression part
	hashNode, ok := node.Expr.(ast.Node)
	if !ok {
		return nil, fmt.Errorf("type expression is not a Node: %T", node.Expr)
	}
	hash := t.TypeChecker.Hasher.HashNode(hashNode)
	typeName := string(hash.ToTypeIdent())

	// Use original name in comment for documentation
	commentName := string(node.Ident)
	if commentName == "" {
		commentName = typeName
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
					Text: fmt.Sprintf("// %s: %s", commentName, node.Expr.String()),
				},
			},
		},
	}, nil
}

func (t *Transformer) getAssertionBaseTypeIdent(assertion *ast.AssertionNode) (*goast.Ident, error) {
	if assertion.BaseType != nil {
		return transformTypeIdent(*assertion.BaseType), nil
	}
	typeNode, err := t.TypeChecker.LookupAssertionType(assertion)
	if err != nil {
		err = fmt.Errorf("failed to lookup assertion type during getAssertionBaseTypeIdent: %w", err)
		log.WithError(err).Error("transforming assertion base type ident failed")
		return nil, err
	}
	return transformTypeIdent(typeNode.Ident), nil
}

// TODO: Improve shape type registration
// This should handle nested shapes better and generate appropriate validation code
func (t *Transformer) defineShapeType(shape *ast.ShapeNode) error {
	// First register all nested shape fields
	if err := t.defineShapeFields(shape); err != nil {
		return err
	}

	// Then register the shape itself
	typeIdent := t.TypeChecker.Hasher.HashNode(shape).ToTypeIdent()

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
			typeIdent := t.TypeChecker.Hasher.HashNode(field.Shape).ToTypeIdent()

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

// getTypeAliasNameForTypeNode returns the hash-based type alias name for a given TypeNode.
// It finds the corresponding TypeDefNode in the type checker and hashes its Expr.
func (t *Transformer) getTypeAliasNameForTypeNode(typeNode ast.TypeNode) (string, error) {
	// If the type is a built-in, return its Go name
	if isGoBuiltinType(string(typeNode.Ident)) || typeNode.Ident == ast.TypeString || typeNode.Ident == ast.TypeInt || typeNode.Ident == ast.TypeFloat || typeNode.Ident == ast.TypeBool || typeNode.Ident == ast.TypeVoid {
		ident := transformTypeIdent(typeNode.Ident)
		if ident != nil {
			return ident.Name, nil
		}
		return string(typeNode.Ident), nil
	}
	// Find the type definition by identifier
	for _, def := range t.TypeChecker.Defs {
		if typeDef, ok := def.(ast.TypeDefNode); ok {
			if typeDef.Ident == typeNode.Ident {
				hashNode, ok := typeDef.Expr.(ast.Node)
				if !ok {
					// fallback: use ident directly
					return string(typeNode.Ident), nil
				}
				hash := t.TypeChecker.Hasher.HashNode(hashNode)
				return string(hash.ToTypeIdent()), nil
			}
		}
	}

	if typeNode.Ident == ast.TypeAssertion {
		assertionType, err := t.TypeChecker.LookupAssertionType(typeNode.Assertion)
		if err != nil {
			// fallback: use ident directly
			return string(typeNode.Ident), nil
		}
		return string(assertionType.Ident), nil
	}
	// fallback: use ident directly
	return string(typeNode.Ident), nil
}
