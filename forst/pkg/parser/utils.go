package parser

import (
	"fmt"
	"forst/pkg/ast"
	"unicode"
)

func unexpectedTokenMessage(token ast.Token, expected string) string {
	return parseErrorMessage(token, fmt.Sprintf("Unexpected token %s, expected %s", token.Value, expected))
}

func parseErrorWithValue(token ast.Token, message string) string {
	return parseErrorMessage(token, fmt.Sprintf("%s. Received: %s", message, token.Value))
}

func parseErrorMessage(token ast.Token, message string) string {
	return fmt.Sprintf(
		"\nParse error in %s:%d:%d (at line %d, column %d):\n"+
			"%s",
		token.Path, token.Line, token.Column, token.Line, token.Column, message,
	)
}

func isCapitalCase(value string) bool {
	return unicode.IsUpper(rune(value[0]))
}
