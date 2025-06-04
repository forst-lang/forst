package parser

import (
	"forst/internal/ast"
)

func (p *Parser) parseBlockStatement() []ast.Node {
	body := []ast.Node{}

	token := p.current()

	switch token.Type {
	case ast.TokenVar:
		if p.context.IsTypeGuard() {
			panic(parseErrorWithValue(token, "Variable declaration not allowed in type guards"))
		}
		varStatement := p.parseVarStatement()
		logParsedNode(varStatement)
		body = append(body, varStatement)
	case ast.TokenEnsure:
		ensureStatement := p.parseEnsureStatement()
		logParsedNode(ensureStatement)
		body = append(body, ensureStatement)
	case ast.TokenReturn:
		if p.context.IsTypeGuard() {
			panic(parseErrorWithValue(token, "Return statement not allowed in type guards"))
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
	case ast.TokenIf:
		ifStatement := p.parseIfStatement()
		logParsedNode(ifStatement)
		body = append(body, ifStatement)
	// case ast.TokenFor:
	// if p.context.IsTypeGuard() {
	// 	panic(parseErrorWithValue(token, "For loop not allowed in type guards"))
	// }
	// 	forStatement := p.parseForStatement()
	// 	logParsedNode(forStatement)
	// 	body = append(body, forStatement)
	// case ast.TokenSwitch:
	// if p.context.IsTypeGuard() {
	// 	panic(parseErrorWithValue(token, "Switch statement not allowed in type guards"))
	// }
	// 	switchStatement := p.parseSwitchStatement()
	// 	logParsedNode(switchStatement)
	// 	body = append(body, switchStatement)
	default:
		expr := p.parseExpression()
		logParsedNode(expr)
		body = append(body, expr)
	}

	return body
}
