package parser

import (
	"forst/internal/ast"
)

type forHeaderKind int

const (
	forHeaderCond forHeaderKind = iota
	forHeaderThree
	forHeaderRange
)

// classifyForHeader peeks ahead (without consuming) to distinguish `for cond`,
// three-clause `for init; cond; post`, and `for ... range`.
func (p *Parser) classifyForHeader() forHeaderKind {
	depth := 0
	semi := 0
	hasRange := false
	for i := 0; ; i++ {
		tok := p.peek(i)
		if tok.Type == ast.TokenEOF {
			if semi > 0 {
				return forHeaderThree
			}
			if hasRange {
				return forHeaderRange
			}
			return forHeaderCond
		}
		switch tok.Type {
		case ast.TokenLParen:
			depth++
		case ast.TokenRParen:
			if depth > 0 {
				depth--
			}
		case ast.TokenSemicolon:
			if depth == 0 {
				semi++
			}
		case ast.TokenRange:
			if depth == 0 {
				hasRange = true
			}
		case ast.TokenLBrace:
			if depth == 0 {
				if semi > 0 {
					return forHeaderThree
				}
				if hasRange {
					return forHeaderRange
				}
				return forHeaderCond
			}
		}
	}
}

func (p *Parser) parseForStatement() ast.Node {
	p.advance() // for
	if p.current().Type == ast.TokenLBrace {
		return &ast.ForNode{
			Body: p.parseBlock(),
		}
	}

	switch p.classifyForHeader() {
	case forHeaderThree:
		return p.parseForThreeClause()
	case forHeaderRange:
		return p.parseForRangeClause()
	default:
		cond := p.parseForConditionExpression()
		body := p.parseBlock()
		return &ast.ForNode{
			Cond: cond,
			Body: body,
		}
	}
}

// parseForConditionExpression parses the condition in `for cond { ... }` without treating
// `ident {` as a shape literal (the `{` begins the loop body).
func (p *Parser) parseForConditionExpression() ast.ExpressionNode {
	if p.current().Type == ast.TokenIdentifier && p.peek().Type == ast.TokenLBrace {
		ident := p.parseIdentifier()
		return ast.VariableNode{Ident: ast.Ident{ID: ident}}
	}
	return p.parseExpression()
}

func (p *Parser) parseForThreeClause() *ast.ForNode {
	var init ast.Node
	if p.current().Type != ast.TokenSemicolon {
		init = p.parseSimpleStatement()
	}
	p.expect(ast.TokenSemicolon)

	var cond ast.ExpressionNode
	if p.current().Type != ast.TokenSemicolon {
		cond = p.parseForConditionExpression()
	}
	p.expect(ast.TokenSemicolon)

	var post ast.Node
	if p.current().Type != ast.TokenLBrace {
		post = p.parseSimpleStatement()
	}

	body := p.parseBlock()
	return &ast.ForNode{
		Init: init,
		Cond: cond,
		Post: post,
		Body: body,
	}
}

func (p *Parser) parseForRangeClause() *ast.ForNode {
	if p.current().Type == ast.TokenRange {
		p.advance()
		x := p.parseRangeExpression()
		body := p.parseBlock()
		return &ast.ForNode{
			IsRange: true,
			RangeX:  x,
			Body:    body,
		}
	}

	first := p.expect(ast.TokenIdentifier)
	key := &ast.Ident{ID: ast.Identifier(first.Value)}

	var val *ast.Ident
	if p.current().Type == ast.TokenComma {
		p.advance()
		second := p.expect(ast.TokenIdentifier)
		val = &ast.Ident{ID: ast.Identifier(second.Value)}
	}

	assignTok := p.current()
	if assignTok.Type != ast.TokenColonEquals && assignTok.Type != ast.TokenEquals {
		p.FailWithParseError(assignTok, "expected := or = before range")
	}
	isShort := assignTok.Type == ast.TokenColonEquals
	p.advance()

	p.expect(ast.TokenRange)
	x := p.parseRangeExpression()
	body := p.parseBlock()

	return &ast.ForNode{
		IsRange:    true,
		RangeX:     x,
		RangeKey:   key,
		RangeValue: val,
		RangeShort: isShort,
		Body:       body,
	}
}

// parseRangeExpression parses the operand of "range" without treating `ident {` as a shape literal
// (the `{` begins the for-body, not a composite literal).
func (p *Parser) parseRangeExpression() ast.ExpressionNode {
	if p.current().Type == ast.TokenIdentifier && p.peek().Type == ast.TokenLBrace {
		ident := p.parseIdentifier()
		return ast.VariableNode{Ident: ast.Ident{ID: ident}}
	}
	return p.parseExpression()
}

func (p *Parser) parseBreakStatement() ast.Node {
	p.advance() // break
	if p.current().Type == ast.TokenIdentifier {
		id := p.expect(ast.TokenIdentifier)
		return &ast.BreakNode{Label: &ast.Ident{ID: ast.Identifier(id.Value)}}
	}
	return &ast.BreakNode{}
}

func (p *Parser) parseContinueStatement() ast.Node {
	p.advance() // continue
	if p.current().Type == ast.TokenIdentifier {
		id := p.expect(ast.TokenIdentifier)
		return &ast.ContinueNode{Label: &ast.Ident{ID: ast.Identifier(id.Value)}}
	}
	return &ast.ContinueNode{}
}
