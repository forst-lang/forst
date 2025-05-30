package typechecker

import (
	"fmt"
	"forst/internal/ast"
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

	// If this is a reference to an existing type, use that type's hash
	if assertion.BaseType != nil {
		if typeDef, exists := tc.Defs[ast.TypeIdent(*assertion.BaseType)]; exists {
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
				}
			}
		}
	}

	return inferredType, nil
}
