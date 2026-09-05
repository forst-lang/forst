package parser

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/diag"
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

// parseReport builds a structured diagnostic message for FailWithReport.
func parseReport(code, title, problem, help string, notes ...string) string {
	return diag.FormatReport(diag.Report{
		Code:    code,
		Title:   title,
		Problem: problem,
		Help:    help,
		Notes:   notes,
	})
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

// looksLikeTypedCompositeOrShapeBody reports whether `{` after a type-like name starts a
// composite/shape (empty or `field: value`) rather than an if/for block body.
func (p *Parser) looksLikeTypedCompositeOrShapeBody() bool {
	if p.current().Type != ast.TokenLBrace {
		return false
	}
	next := p.peek()
	if next.Type == ast.TokenRBrace {
		return true
	}
	// Shape / keyed composite: member name then colon.
	switch next.Type {
	case ast.TokenIdentifier, ast.TokenString, ast.TokenInt, ast.TokenFloat, ast.TokenBool:
		return p.peek(2).Type == ast.TokenColon
	default:
		return false
	}
}
