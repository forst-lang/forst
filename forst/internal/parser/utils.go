package parser

import (
	"fmt"
	"forst/internal/ast"
	"unicode"

	log "github.com/sirupsen/logrus"
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

func logParsedNode(node ast.Node) {
	logParsedNodeWithMessage(node, "Parsed node")
}

func logParsedNodeWithMessage(node ast.Node, message string) {
	log.WithField("node", node).Trace(message)
}

func isCapitalCase(value string) bool {
	return unicode.IsUpper(rune(value[0]))
}

func isParenthesis(token ast.Token) bool {
	return token.Type == ast.TokenLParen || token.Type == ast.TokenRParen
}
