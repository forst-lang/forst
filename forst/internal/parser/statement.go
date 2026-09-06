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
			p.FailWithReport(token, "typeguard-forbidden-var", "variable declaration not allowed in type guards",
				"Variable declaration not allowed in type guards.",
				"declare variables outside the guard body")
		}
		varStatement := p.parseVarStatement()
		p.logParsedNode(varStatement)
		body = append(body, varStatement)
	case ast.TokenEnsure:
		ensureStatement := p.parseEnsureStatement()
		p.logParsedNode(ensureStatement)
		body = append(body, ensureStatement)
	case ast.TokenUse:
		if p.context.IsTypeGuard() {
			p.FailWithReport(token, "typeguard-forbidden-use", "use statement not allowed in type guards",
				"use statement not allowed in type guards.",
				"declare provider bindings outside the guard")
		}
		useStatement := p.parseUseStatement()
		p.logParsedNode(useStatement)
		body = append(body, useStatement)
	case ast.TokenWith:
		if p.context.IsTypeGuard() {
			p.FailWithReport(token, "typeguard-forbidden-with", "with statement not allowed in type guards",
				"with statement not allowed in type guards.",
				"use with wiring outside the guard body")
		}
		withStatement := p.parseWithStatement()
		p.logParsedNode(withStatement)
		body = append(body, withStatement)
	case ast.TokenReturn:
		if p.context.IsTypeGuard() {
			p.FailWithReport(token, "typeguard-forbidden-return", "return statement not allowed in type guards",
				"Return statement not allowed in type guards.",
				"type guards refine types; use ensure instead of return")
		}
		returnStatement := p.parseReturnStatement()
		p.logParsedNode(returnStatement)
		body = append(body, returnStatement)
	case ast.TokenGoto:
		if p.context.IsTypeGuard() {
			p.FailWithReport(token, "typeguard-forbidden-goto", "goto not allowed in type guards",
				"goto not allowed in type guards.",
				"remove goto from the guard body")
		}
		g := p.parseGotoStatement()
		p.logParsedNode(g)
		body = append(body, g)
	case ast.TokenIdentifier:
		next := p.peek()
		if next.Type == ast.TokenColon && p.looksLikeLabeledStatement() {
			if p.context.IsTypeGuard() {
				p.FailWithReport(token, "typeguard-forbidden-label", "labeled statements not allowed in type guards",
					"labeled statements not allowed in type guards.",
					"remove labels from the guard body")
			}
			labeled := p.parseLabeledStatement()
			p.logParsedNode(labeled)
			body = append(body, labeled)
			break
		}
		if next.Type == ast.TokenComma {
			assignment := p.parseMultipleAssignment()
			p.logParsedNode(assignment)
			body = append(body, assignment)
			break
		}
		if next.Type == ast.TokenPlusPlus || next.Type == ast.TokenMinusMinus {
			incDec := p.parseIncDecStmt()
			p.logParsedNode(incDec)
			body = append(body, incDec)
			break
		}
		if next.Type == ast.TokenPlus && p.peek(1).Type == ast.TokenPlus {
			incDec := p.parseIncDecStmt()
			p.logParsedNode(incDec)
			body = append(body, incDec)
			break
		}
		if next.Type == ast.TokenMinus && p.peek(1).Type == ast.TokenMinus {
			incDec := p.parseIncDecStmt()
			p.logParsedNode(incDec)
			body = append(body, incDec)
			break
		}
		if next.Type == ast.TokenArrow {
			send := p.parseSendStmt()
			p.logParsedNode(send)
			body = append(body, send)
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
		if ast.IsAssignmentOperatorToken(p.current()) {
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
	case ast.TokenSwitch:
		if p.context.IsTypeGuard() {
			p.FailWithReport(token, "typeguard-forbidden-switch", "switch not allowed in type guards",
				"switch not allowed in type guards.",
				"use `if x is Foo() { ... }` narrowing instead")
		}
		sw := p.parseSwitchStatement()
		p.logParsedNode(sw)
		body = append(body, sw)
	case ast.TokenFor:
		if p.context.IsTypeGuard() {
			p.FailWithReport(token, "typeguard-forbidden-for", "for loop not allowed in type guards",
				"For loop not allowed in type guards.",
				"move loops outside the guard body")
		}
		forStatement := p.parseForStatement()
		p.logParsedNode(forStatement)
		body = append(body, forStatement)
	case ast.TokenBreak:
		if p.context.IsTypeGuard() {
			p.FailWithReport(token, "typeguard-forbidden-break", "break not allowed in type guards",
				"Break not allowed in type guards.",
				"remove break from the guard body")
		}
		br := p.parseBreakStatement()
		p.logParsedNode(br)
		body = append(body, br)
	case ast.TokenContinue:
		if p.context.IsTypeGuard() {
			p.FailWithReport(token, "typeguard-forbidden-continue", "continue not allowed in type guards",
				"Continue not allowed in type guards.",
				"remove continue from the guard body")
		}
		cont := p.parseContinueStatement()
		p.logParsedNode(cont)
		body = append(body, cont)
	case ast.TokenDefer:
		if p.context.IsTypeGuard() {
			p.FailWithReport(token, "typeguard-forbidden-defer", "defer not allowed in type guards",
				"defer not allowed in type guards.",
				"move defer outside the guard body")
		}
		d := p.parseDeferStatement()
		p.logParsedNode(d)
		body = append(body, d)
	case ast.TokenGo:
		if p.context.IsTypeGuard() {
			p.FailWithReport(token, "typeguard-forbidden-go", "go not allowed in type guards",
				"go not allowed in type guards.",
				"move go statements outside the guard body")
		}
		g := p.parseGoStatement()
		p.logParsedNode(g)
		body = append(body, g)
	case ast.TokenStar:
		saved := p.currentIndex
		if lhs, ok := p.parseDerefAssignableExpr(); ok {
			if ast.IsAssignmentOperatorToken(p.current()) {
				assign := p.finishAssignment(lhs)
				p.logParsedNode(assign)
				body = append(body, assign)
				break
			}
		}
		p.currentIndex = saved
		expr := p.parseExpression()
		p.logParsedNode(expr)
		body = append(body, expr)
	default:
		expr := p.parseExpression()
		p.logParsedNode(expr)
		body = append(body, expr)
	}

	return body
}
