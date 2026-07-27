package parser

import "forst/internal/ast"

// looksLikeLabeledStatement reports whether current ident + ':' starts a labeled
// statement rather than a typed var decl (`x: Type =` / `x: Type :=`).
// Caller must be positioned on the identifier with peek() == colon.
func (p *Parser) looksLikeLabeledStatement() bool {
	if p.current().Type != ast.TokenIdentifier || p.peek().Type != ast.TokenColon {
		return false
	}
	afterColon := p.peek(2)
	switch afterColon.Type {
	case ast.TokenFor, ast.TokenIf, ast.TokenSwitch, ast.TokenReturn, ast.TokenBreak,
		ast.TokenContinue, ast.TokenGoto, ast.TokenDefer, ast.TokenGo, ast.TokenEnsure,
		ast.TokenVar, ast.TokenUse, ast.TokenWith:
		return true
	}

	// Typed assign: after colon, a type then = or :=.
	saved := p.currentIndex
	p.advance() // ident
	p.advance() // colon
	isTyped := p.tryParseTypeThenAssignOp()
	p.currentIndex = saved
	return !isTyped
}

// tryParseTypeThenAssignOp reports whether the tokens at the current position form
// a type followed by = or :=. Uses recover because parseType panics on invalid input.
func (p *Parser) tryParseTypeThenAssignOp() (ok bool) {
	saved := p.currentIndex
	defer func() {
		if recover() != nil {
			ok = false
			p.currentIndex = saved
		}
	}()
	_ = p.parseType(TypeIdentOpts{AllowLowercaseTypes: false})
	tok := p.current()
	if tok.Type == ast.TokenEquals || tok.Type == ast.TokenColonEquals {
		return true
	}
	p.currentIndex = saved
	return false
}

func (p *Parser) parseGotoStatement() ast.Node {
	p.advance() // goto
	if p.current().Type != ast.TokenIdentifier {
		p.FailWithParseError(p.current(), "expected label after goto")
	}
	id := p.expect(ast.TokenIdentifier)
	return &ast.GotoNode{Label: &ast.Ident{ID: ast.Identifier(id.Value), Span: ast.SpanFromToken(id)}}
}

func (p *Parser) parseLabeledStatement() ast.Node {
	idTok := p.expect(ast.TokenIdentifier)
	p.expect(ast.TokenColon)
	label := &ast.Ident{ID: ast.Identifier(idTok.Value), Span: ast.SpanFromToken(idTok)}

	stmts := p.parseBlockStatement()
	if len(stmts) != 1 {
		p.FailWithParseError(idTok, "labeled statement requires exactly one statement after the label")
	}
	stmt := stmts[0]
	if fn, ok := stmt.(*ast.ForNode); ok {
		fn.Label = label
		return fn
	}
	return &ast.LabeledStmtNode{Label: label, Stmt: stmt}
}
