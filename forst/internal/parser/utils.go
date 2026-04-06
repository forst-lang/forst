package parser

import (
	"fmt"
	"forst/internal/ast"
	"strings"
	"unicode"
)

func unexpectedTokenMessage(token ast.Token, expected string) string {
	return parseErrorMessage(token, fmt.Sprintf("Unexpected token %s, expected %s", token.Value, expected))
}

func parseErrorMessage(token ast.Token, message string) string {
	return fmt.Sprintf(
		"\nParse error in %s:%d:%d (at line %d, column %d):\n"+
			"%s",
		token.FileID, token.Line, token.Column, token.Line, token.Column, message,
	)
}

func (p *Parser) logParsedNode(node ast.Node) {
	p.logParsedNodeWithMessage(node, "Parsed node")
}

func (p *Parser) logParsedNodeWithMessage(node ast.Node, message string) {
	p.log.WithField("node", node).Trace(message)
}

func isCapitalCase(value string) bool {
	if value == "" {
		return false
	}
	return unicode.IsUpper(rune(value[0]))
}

// isShapeLiteralTypePrefix reports whether Identifier "{" should be parsed as a typed
// composite literal (TypeName { ... }). Lowercase identifiers are left as variables so
// expressions like `x == c {` can end the comparison before `{` starts a block.
func isShapeLiteralTypePrefix(ident string) bool {
	if ident == "" {
		return false
	}
	i := strings.LastIndex(ident, ".")
	if i >= 0 {
		ident = ident[i+1:]
	}
	return isCapitalCase(ident)
}
