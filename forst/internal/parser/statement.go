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
			p.FailWithParseError(token, "Variable declaration not allowed in type guards")
		}
		varStatement := p.parseVarStatement()
		p.logParsedNode(varStatement)
		body = append(body, varStatement)
	case ast.TokenEnsure:
		ensureStatement := p.parseEnsureStatement()
		p.logParsedNode(ensureStatement)
		body = append(body, ensureStatement)
	case ast.TokenReturn:
		if p.context.IsTypeGuard() {
			p.FailWithParseError(token, "Return statement not allowed in type guards")
		}
		returnStatement := p.parseReturnStatement()
		p.logParsedNode(returnStatement)
		body = append(body, returnStatement)
	case ast.TokenIdentifier:
		next := p.peek()
		if next.Type == ast.TokenComma {
			assignment := p.parseMultipleAssignment()
			p.logParsedNode(assignment)
			body = append(body, assignment)
			break
		}
		// x = …, x := …, x: T = …, xs[i] = … (assignable expression, then = or := or : Type …)
		saved := p.currentIndex
		lhs := p.parseAssignableExpr()
		if _, ok := lhs.(ast.IndexExpressionNode); ok && p.current().Type == ast.TokenColon {
			p.currentIndex = saved
			expr := p.parseExpression()
			p.logParsedNode(expr)
			body = append(body, expr)
			break
		}
		if p.current().Type == ast.TokenEquals || p.current().Type == ast.TokenColonEquals {
			assign := p.finishAssignment(lhs)
			p.logParsedNode(assign)
			body = append(body, assign)
			break
		}
		if _, ok := lhs.(ast.VariableNode); ok && p.current().Type == ast.TokenColon {
			assign := p.finishAssignment(lhs)
			p.logParsedNode(assign)
			body = append(body, assign)
			break
		}
		p.currentIndex = saved
		expr := p.parseExpression()
		p.logParsedNode(expr)
		body = append(body, expr)
	case ast.TokenIf:
		ifStatement := p.parseIfStatement()
		p.logParsedNode(ifStatement)
		body = append(body, ifStatement)
	case ast.TokenFor:
		if p.context.IsTypeGuard() {
			p.FailWithParseError(token, "For loop not allowed in type guards")
		}
		forStatement := p.parseForStatement()
		p.logParsedNode(forStatement)
		body = append(body, forStatement)
	case ast.TokenBreak:
		if p.context.IsTypeGuard() {
			p.FailWithParseError(token, "Break not allowed in type guards")
		}
		br := p.parseBreakStatement()
		p.logParsedNode(br)
		body = append(body, br)
	case ast.TokenContinue:
		if p.context.IsTypeGuard() {
			p.FailWithParseError(token, "Continue not allowed in type guards")
		}
		cont := p.parseContinueStatement()
		p.logParsedNode(cont)
		body = append(body, cont)
	case ast.TokenDefer:
		if p.context.IsTypeGuard() {
			p.FailWithParseError(token, "defer not allowed in type guards")
		}
		d := p.parseDeferStatement()
		p.logParsedNode(d)
		body = append(body, d)
	case ast.TokenGo:
		if p.context.IsTypeGuard() {
			p.FailWithParseError(token, "go not allowed in type guards")
		}
		g := p.parseGoStatement()
		p.logParsedNode(g)
		body = append(body, g)
	default:
		expr := p.parseExpression()
		p.logParsedNode(expr)
		body = append(body, expr)
	}

	return body
}
