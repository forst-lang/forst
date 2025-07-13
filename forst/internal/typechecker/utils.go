package typechecker

import (
	"fmt"
	"forst/internal/ast"
	"strings"
)

func formatTypeList(types []ast.TypeNode) string {
	if len(types) == 0 {
		return "()"
	}

	formatted := make([]string, len(types))
	for i, typ := range types {
		formatted[i] = typ.String()
	}
	return strings.Join(formatted, ", ")
}

func failWithTypeMismatch(fn ast.FunctionNode, inferred []ast.TypeNode, parsed []ast.TypeNode, prefix string) error {
	return fmt.Errorf("%s: inferred return type %v of function %s does not match parsed return type %v", prefix, formatTypeList(inferred), fn.Ident.ID, formatTypeList(parsed))
}

// Ensures that the first type matches the expected type, otherwise returns an error
func ensureMatching(tc *TypeChecker, fn ast.FunctionNode, actual []ast.TypeNode, expected []ast.TypeNode, prefix string) ([]ast.TypeNode, error) {
	if len(expected) == 0 {
		// If the expected type is implicit, we have nothing to check against
		return actual, nil
	}

	if len(actual) != len(expected) {
		return actual, failWithTypeMismatch(fn, actual, expected, fmt.Sprintf("%s: Type arity mismatch", prefix))
	}

	for i := range expected {
		if !tc.IsTypeCompatible(actual[i], expected[i]) {
			return actual, failWithTypeMismatch(fn, actual, expected, fmt.Sprintf("%s: Type mismatch", prefix))
		}
	}

	return actual, nil
}

func typecheckErrorMessageWithNode(node *ast.Node, message string) string {
	return fmt.Sprintf(
		"\nTypecheck error at %s:\n"+
			"%s",
		(*node).String(), message,
	)
}

func typecheckError(message string) string {
	return fmt.Sprintf(
		"\nTypecheck error:\n%s",
		message,
	)
}

// IsShapeSuperset returns true if shape a has all fields of shape b (with compatible types), possibly with extras.
// This uses the existing shape merging logic to properly handle nested assertions and shape guards.
func (tc *TypeChecker) IsShapeSuperset(a, b ast.ShapeNode) bool {
	// For now, use the existing structural compatibility check
	// TODO: Enhance this to use resolveShapeFieldsFromAssertion for complex cases
	return tc.shapesAreStructurallyIdentical(a, b)
}

// IsShapeCompatibleWithNamedType checks if a shape is compatible with a named type,
// taking into account the full shape merging logic including nested assertions and constraints.
func (tc *TypeChecker) IsShapeCompatibleWithNamedType(shape ast.ShapeNode, namedType ast.TypeIdent) bool {
	// Get the named type definition
	def, exists := tc.Defs[namedType]
	if !exists {
		return false
	}

	// If the named type is a shape definition, compare directly
	if typeDef, ok := def.(ast.TypeDefNode); ok {
		if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
			return tc.shapesAreStructurallyIdentical(shape, shapeExpr.Shape)
		}
	}

	// If the named type is an assertion, resolve its fields and compare
	if typeDef, ok := def.(ast.TypeDefNode); ok {
		if assertionExpr, ok := typeDef.Expr.(ast.TypeDefAssertionExpr); ok {
			if assertionExpr.Assertion != nil {
				// Use the existing shape merging logic
				mergedFields := tc.resolveShapeFieldsFromAssertion(assertionExpr.Assertion)
				mergedShape := ast.ShapeNode{Fields: mergedFields}
				return tc.shapesAreStructurallyIdentical(shape, mergedShape)
			}
		}
	}

	return false
}

// RegisterHashBasedType registers a hash-based type and its fields in the typechecker.
func RegisterHashBasedType(tc *TypeChecker, typeIdent ast.TypeIdent, fields map[string]ast.ShapeFieldNode) {
	// Only register if not already present
	if _, exists := tc.Defs[typeIdent]; !exists {
		tc.Defs[typeIdent] = ast.TypeDefNode{
			Ident: typeIdent,
			Expr: ast.TypeDefShapeExpr{
				Shape: ast.ShapeNode{
					Fields: fields,
				},
			},
		}
	}
}
