package typechecker

import (
	"fmt"
	"forst/internal/ast"

	log "github.com/sirupsen/logrus"
)

// TODO: Improve assertion type inference
// This should handle:
// 1. Complex constraints
// 2. Nested assertions
// 3. Type aliases
// 4. Generic types
func (tc *TypeChecker) inferAssertionType(assertion *ast.AssertionNode, requireInferred bool) ([]ast.TypeNode, error) {
	// Check if we've already inferred this assertion's type
	existingTypes, err := tc.LookupInferredType(assertion, requireInferred)
	if err != nil {
		return nil, err
	}

	if existingTypes != nil {
		return existingTypes, nil
	}

	// If this is a type guard application, construct a shape type that includes the fields introduced by the type guard
	if assertion.BaseType != nil {
		if typeDef, exists := tc.Defs[ast.TypeIdent(*assertion.BaseType)]; exists {
			// Check if this is a type guard application (i.e., constraints present)
			if len(assertion.Constraints) > 0 {
				fields := map[string]ast.ShapeFieldNode{}
				for _, constraint := range assertion.Constraints {
					if guardDef, exists := tc.Defs[ast.TypeIdent(constraint.Name)]; exists {
						if guardNode, ok := guardDef.(ast.TypeGuardNode); ok {
							if len(guardNode.Params) > 0 && len(constraint.Args) > 0 {
								param := guardNode.Params[0]
								paramName := param.GetIdent()
								argType := constraint.Args[0]
								if argType.Type != nil {
									fields[paramName] = ast.ShapeFieldNode{
										Assertion: &ast.AssertionNode{
											BaseType: &argType.Type.Ident,
										},
									}
								}
							}
						}
					}
				}
				// Store the shape in tc.Defs under a unique type identifier
				shape := ast.ShapeNode{Fields: fields}
				hash := tc.Hasher.HashNode(assertion)
				typeIdent := hash.ToTypeIdent()
				tc.Defs[typeIdent] = shape
				log.WithFields(log.Fields{
					"typeIdent": typeIdent,
					"assertion": assertion.String(),
					"fields":    fields,
				}).Tracef("[inferAssertionType] Stored shape type with fields")
				shapeType := ast.TypeNode{Ident: typeIdent}
				tc.storeInferredType(assertion, []ast.TypeNode{shapeType})
				return []ast.TypeNode{shapeType}, nil
			}
			// Otherwise, use the type's hash as before
			hash := tc.Hasher.HashNode(typeDef)
			typeIdent := hash.ToTypeIdent()
			typeNode := ast.TypeNode{
				Ident:     typeIdent,
				Assertion: assertion,
			}
			inferredType := []ast.TypeNode{typeNode}
			tc.storeInferredType(assertion, inferredType)
			return inferredType, nil
		}
	}

	// Otherwise create a new type alias
	hash := tc.Hasher.HashNode(assertion)
	typeIdent := hash.ToTypeIdent()
	typeNode := ast.TypeNode{
		Ident:     typeIdent,
		Assertion: assertion,
	}
	inferredType := []ast.TypeNode{typeNode}
	tc.storeInferredType(assertion, inferredType)

	// Process each constraint in the assertion
	tc.registerType(ast.TypeDefNode{
		Ident: typeIdent,
		Expr: ast.TypeDefAssertionExpr{
			Assertion: assertion,
		},
	})

	if assertion.BaseType != nil && (*assertion.BaseType == "trpc.Mutation" || *assertion.BaseType == "trpc.Query") {
		for _, constraint := range assertion.Constraints {
			switch constraint.Name {
			case "Input":
				if len(constraint.Args) != 1 {
					return nil, fmt.Errorf("input constraint must have exactly one argument")
				}
				arg := constraint.Args[0]
				if arg.Shape != nil {
					if _, err := tc.inferShapeType(arg.Shape); err != nil {
						return nil, fmt.Errorf("failed to infer shape type: %w", err)
					}
				} else if arg.Type != nil {
					// If the argument is a type, just check that it exists
					if _, exists := tc.Defs[arg.Type.Ident]; !exists {
						return nil, fmt.Errorf("type argument %s not defined", arg.Type.Ident)
					}
				} else if arg.Value != nil {
					// Optionally handle value arguments if needed
					// (not expected for shape guards)
				}
			}
		}
	}

	return inferredType, nil
}
