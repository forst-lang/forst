package parser

import (
	"forst/pkg/ast"
)

func (p *Parser) parseBlockStatement(blockContext *BlockContext) []ast.Node {
	body := []ast.Node{}

	token := p.current()

	if token.Type == ast.TokenEnsure {
		ensureStatement := p.parseEnsureStatement()
		logParsedNode(ensureStatement)
		body = append(body, ensureStatement)
	} else if token.Type == ast.TokenReturn {
		if !blockContext.AllowReturn {
			panic(parseErrorWithValue(token, "Return statement not allowed in this context"))
		}
		returnStatement := p.parseReturnStatement()
		logParsedNode(returnStatement)
		body = append(body, returnStatement)
	} else if token.Type == ast.TokenIdentifier {
		next := p.peek()
		// Look ahead to see if this is a function call or assignment
		if next.Type == ast.TokenLParen || next.Type == ast.TokenDot {
			// Function call
			expr := p.parseExpression()
			logParsedNode(expr)
			body = append(body, expr)
		} else if next.Type == ast.TokenComma {
			// Multiple assignment
			assignment := p.parseMultipleAssignment()
			logParsedNode(assignment)
			body = append(body, assignment)
		} else if next.Type == ast.TokenColonEquals || next.Type == ast.TokenEquals {
			// Single assignment
			assignment := p.parseAssignment()
			logParsedNode(assignment)
			body = append(body, assignment)
		} else {
			panic(parseErrorWithValue(token, "Expected function call or assignment after identifier"))
		}
	} else {
		panic(parseErrorWithValue(token, "Unexpected token in function body"))
	}

	return body
}
