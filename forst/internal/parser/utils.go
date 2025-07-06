package parser

import (
	"fmt"
	"forst/internal/ast"
	"unicode"
)

func unexpectedTokenMessage(token ast.Token, expected string) string {
	return parseErrorMessage(token, fmt.Sprintf("Unexpected token %s, expected %s", token.Value, expected))
}

func parseErrorMessage(token ast.Token, message string) string {
	return fmt.Sprintf(
		"\nParse error in %s:%d:%d (at line %d, column %d):\n"+
			"%s",
		token.Path, token.Line, token.Column, token.Line, token.Column, message,
	)
}

func (p *Parser) logParsedNode(node ast.Node) {
	p.logParsedNodeWithMessage(node, "Parsed node")
}

func (p *Parser) logParsedNodeWithMessage(node ast.Node, message string) {
	p.log.WithField("node", node).Trace(message)
}

func isCapitalCase(value string) bool {
	return unicode.IsUpper(rune(value[0]))
}
