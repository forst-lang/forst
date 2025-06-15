package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"

	logrus "github.com/sirupsen/logrus"
)

// TODO: Implement binary type expressions
// This should handle both conjunction (&) and disjunction (|) operators
// and generate appropriate validation code
func (t *Transformer) transformTypeDefExpr(expr ast.TypeDefExpr) (*goast.Expr, error) {
	switch e := expr.(type) {
	case ast.TypeDefAssertionExpr:
		baseTypeIdent, err := t.getAssertionBaseTypeIdent(e.Assertion)
		if err != nil {
			err = fmt.Errorf("failed to get assertion base type ident during transformation: %w", err)
			t.log.WithFields(logrus.Fields{
				"function": "transformTypeDefExpr",
				"expr":     expr,
			}).WithError(err).Error("transforming assertion base type ident failed")
			return nil, err
		}
		if baseTypeIdent.Name == "trpc.Mutation" || baseTypeIdent.Name == "trpc.Query" {
			fields := []*goast.Field{
				{
					Names: []*goast.Ident{goast.NewIdent("ctx")},
					Type:  &goast.StructType{Fields: &goast.FieldList{}},
				},
			}

			for _, constraint := range e.Assertion.Constraints {
				if constraint.Name == "Input" && len(constraint.Args) > 0 {
					arg := constraint.Args[0]
					if shape := arg.Shape; shape != nil {
						expr, err := t.transformShapeType(shape)
						if err != nil {
							err = fmt.Errorf("failed to transform shape type during transformation: %w", err)
							t.log.WithFields(logrus.Fields{
								"function": "transformTypeDefExpr",
								"expr":     expr,
							}).WithError(err).Error("transforming shape type failed")
							return nil, err
						}
						inputField := goast.Field{
							Names: []*goast.Ident{goast.NewIdent("input")},
							Type:  *expr,
						}
						fields = append(fields, &inputField)
					}
				}
			}

			result := goast.StructType{
				Fields: &goast.FieldList{
					List: fields,
				},
			}
			var expr goast.Expr = &result
			return &expr, nil
		}

		// For primitive types, use them directly
		if isGoBuiltinType(baseTypeIdent.Name) {
			var result goast.Expr = baseTypeIdent
			return &result, nil
		}

		// Use hash-based type alias for user-defined types
		hash, err := t.TypeChecker.Hasher.HashNode(e)
		if err != nil {
			err = fmt.Errorf("failed to hash type def expr during transformation: %w", err)
			t.log.WithFields(logrus.Fields{
				"function": "transformTypeDefExpr",
				"expr":     expr,
			}).WithError(err).Error("transforming type def expr failed")
			return nil, err
		}
		typeAliasName := hash.ToTypeIdent()
		var result goast.Expr = goast.NewIdent(string(typeAliasName))
		return &result, nil
	case *ast.TypeDefAssertionExpr:
		// Handle pointer by dereferencing and reusing value logic
		return t.transformTypeDefExpr(*e)
	case ast.TypeDefShapeExpr:
		shape := e.Shape
		expr, err := t.transformShapeType(&shape)
		if err != nil {
			err = fmt.Errorf("failed to transform shape type during transformation: %w", err)
			t.log.WithFields(logrus.Fields{
				"function": "transformTypeDefExpr",
				"expr":     expr,
			}).WithError(err).Error("transforming shape type failed")
			return nil, err
		}
		return expr, nil
	case ast.TypeDefBinaryExpr:
		// binaryExpr := expr.(ast.TypeDefBinaryExpr)
		// if binaryExpr.IsConjunction() {
		// 	return &goast.InterfaceType{
		// 		Methods: &goast.FieldList{
		// 			List: []*goast.Field{
		// 				{Type: *t.transformTypeDefExpr(binaryExpr.Left)},
		// 				{Type: *t.transformTypeDefExpr(binaryExpr.Right)},
		// 			},
		// 		},
		// 	}
		// } else if binaryExpr.IsDisjunction() {
		// 	return &goast.InterfaceType{
		// 		Methods: &goast.FieldList{
		// 			List: []*goast.Field{
		// 				{Type: *t.transformTypeDefExpr(binaryExpr.Left)},
		// 				{Type: *t.transformTypeDefExpr(binaryExpr.Right)},
		// 			},
		// 		},
		// 	}
		// }
		ident := goast.NewIdent("string")
		var result goast.Expr = ident
		return &result, nil
	default:
		err := fmt.Errorf("unknown type def expr: %T", expr)
		t.log.WithFields(logrus.Fields{
			"function": "transformTypeDefExpr",
			"expr":     expr,
		}).WithError(err).Error("transforming type def expr failed")
		return nil, err
	}
}
