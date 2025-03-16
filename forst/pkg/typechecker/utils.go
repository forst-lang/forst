package typechecker

import (
	"fmt"
	"forst/pkg/ast"
)

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
