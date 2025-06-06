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
func (tc *TypeChecker) inferAssertionType(assertion *ast.AssertionNode, isTypeGuard bool) ([]ast.TypeNode, error) {
	log.Tracef("[inferAssertionType] Inferring type for assertion: %+v", assertion)

	// Check if we've already inferred this assertion's type
	existingTypes, err := tc.LookupInferredType(assertion, isTypeGuard)
	if err != nil {
		return nil, err
	}

	if existingTypes != nil {
		return existingTypes, nil
	}

	// If this is a type guard application (i.e., constraints present)
	if len(assertion.Constraints) > 0 {
		// Start with an empty shape
		fields := map[string]ast.ShapeFieldNode{}

		// First, if there's a base type, get its fields
		if assertion.BaseType != nil {
			if def, exists := tc.Defs[*assertion.BaseType]; exists {
				if typeDef, ok := def.(ast.TypeDefNode); ok {
					if baseAssertionExpr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
						if baseAssertionExpr.Assertion != nil {
							// Recursively get fields from base type
							baseFields := tc.resolveMergedShapeFields(baseAssertionExpr.Assertion)
							for k, v := range baseFields {
								fields[k] = v
							}
						}
					}
				}
			}
		}

		// Then process each constraint in order
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
						} else if argType.Shape != nil {
							// Handle shape arguments
							fields[paramName] = ast.ShapeFieldNode{
								Shape: argType.Shape,
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
		}).Debugf("[inferAssertionType] Stored shape type with fields")
		shapeType := ast.TypeNode{Ident: typeIdent}
		tc.storeInferredType(assertion, []ast.TypeNode{shapeType})
		return []ast.TypeNode{shapeType}, nil
	}

	// If this is a type guard definition
	if isTypeGuard {
		// For type guards, we need to store the shape type with its fields
		if len(assertion.Constraints) > 0 {
			fields := map[string]ast.ShapeFieldNode{}
			for _, constraint := range assertion.Constraints {
				if constraint.Name == "Match" && len(constraint.Args) > 0 {
					if shapeArg := constraint.Args[0].Shape; shapeArg != nil {
						for name, field := range shapeArg.Fields {
							fields[name] = field
						}
					}
				}
			}
			shape := ast.ShapeNode{Fields: fields}
			hash := tc.Hasher.HashNode(assertion)
			typeIdent := hash.ToTypeIdent()
			tc.Defs[typeIdent] = shape
			log.WithFields(log.Fields{
				"typeIdent": typeIdent,
				"assertion": assertion.String(),
				"fields":    fields,
			}).Debugf("[inferAssertionType] Stored shape type for type guard")
			shapeType := ast.TypeNode{Ident: typeIdent}
			tc.storeInferredType(assertion, []ast.TypeNode{shapeType})
			return []ast.TypeNode{shapeType}, nil
		}
	}

	// Otherwise, use the type's hash as before
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
