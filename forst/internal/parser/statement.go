package parser

import (
	"forst/internal/ast"
)

func (p *Parser) parseBlockStatement(blockContext *BlockContext) []ast.Node {
	body := []ast.Node{}

	token := p.current()

	switch token.Type {
	case ast.TokenEnsure:
		ensureStatement := p.parseEnsureStatement()
		logParsedNode(ensureStatement)
		body = append(body, ensureStatement)
	case ast.TokenReturn:
		if !blockContext.AllowReturn {
			panic(parseErrorWithValue(token, "Return statement not allowed in this context"))
		}
		returnStatement := p.parseReturnStatement()
		logParsedNode(returnStatement)
		body = append(body, returnStatement)
	case ast.TokenIdentifier:
		next := p.peek()
		if next.Type == ast.TokenComma {
			assignment := p.parseMultipleAssignment()
			logParsedNode(assignment)
			body = append(body, assignment)
		} else if next.Type == ast.TokenColonEquals || next.Type == ast.TokenEquals {
			assignment := p.parseAssignment()
			logParsedNode(assignment)
			body = append(body, assignment)
		} else if next.Type == ast.TokenColon && (p.peek(2).Type == ast.TokenIdentifier || p.peek(2).Type == ast.TokenString || p.peek(2).Type == ast.TokenInt || p.peek(2).Type == ast.TokenFloat || p.peek(2).Type == ast.TokenBool || p.peek(2).Type == ast.TokenVoid) && (p.peek(3).Type == ast.TokenColonEquals || p.peek(3).Type == ast.TokenEquals) {
			// identifier: Type := ...
			assignment := p.parseAssignment()
			logParsedNode(assignment)
			body = append(body, assignment)
		} else {
			expr := p.parseExpression()
			logParsedNode(expr)
			body = append(body, expr)
		}
	default:
		expr := p.parseExpression()
		logParsedNode(expr)
		body = append(body, expr)
	}

	return body
}
