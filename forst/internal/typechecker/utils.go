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
func ensureMatching(fn ast.FunctionNode, actual []ast.TypeNode, expected []ast.TypeNode, prefix string) ([]ast.TypeNode, error) {
	if len(expected) == 0 {
		// If the expected type is implicit, we have nothing to check against
		return actual, nil
	}

	if len(actual) != len(expected) {
		return actual, failWithTypeMismatch(fn, actual, expected, fmt.Sprintf("%s: Type arity mismatch", prefix))
	}

	for i := range expected {
		if actual[i].Ident != expected[i].Ident {
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
