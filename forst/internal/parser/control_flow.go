package parser

import (
	"forst/internal/ast"
)

func (p *Parser) parseIfStatement() ast.Node {
	p.advance() // consume if

	// Parse (optional) initialization statement
	var init ast.Node
	var condition ast.ExpressionNode
	switch p.current().Type {
	case ast.TokenVar:
		// Handle var declaration
		p.advance() // consume var
		init = p.parseVarDeclaration()
		if p.current().Type == ast.TokenSemicolon {
			p.advance() // Consume semicolon
		} else {
			p.FailWithParseError(p.current(), "Expected semicolon after var declaration")
		}
	case ast.TokenIdentifier:
		next := p.peek()
		if next.Type == ast.TokenColonEquals || next.Type == ast.TokenEquals {
			// Handle assignment or short var decl
			init = p.parseAssignment()
		} else if next.Type == ast.TokenPlusPlus || next.Type == ast.TokenMinusMinus {
			// Handle inc/dec statement
			init = p.parseIncDecStmt()
		} else if next.Type == ast.TokenArrow {
			// Handle send statement
			init = p.parseSendStmt()
		}

		if init != nil {
			// Expect semicolon after SimpleStmt
			if p.current().Type == ast.TokenSemicolon {
				p.advance() // Consume semicolon
			} else {
				p.FailWithParseError(p.current(), "Expected semicolon after initialization statement")
			}
		}
	}

	// Parse required condition expression
	condition = p.parseExpression()

	if p.current().Type != ast.TokenLBrace {
		p.FailWithParseError(p.current(), "Expected { after if condition")
	}

	// Parse then block
	body := p.parseBlock()

	elseIfs := []ast.ElseIfNode{}
	var elseNode *ast.ElseBlockNode

	for p.current().Type == ast.TokenElse {
		p.advance() // consume else
		if p.current().Type == ast.TokenIf {
			p.advance() // consume if
			elseIfCondition := p.parseExpression()
			if p.current().Type != ast.TokenLBrace {
				p.FailWithParseError(p.current(), "Expected { after else-if condition")
			}
			elseIfBody := p.parseBlock()
			elseIfs = append(elseIfs, ast.ElseIfNode{
				Condition: elseIfCondition,
				Body:      elseIfBody,
			})
		} else {
			// This is a regular else block
			elseBlock := p.parseBlock()
			elseNode = &ast.ElseBlockNode{
				Body: elseBlock,
			}
			break // Exit the loop after finding an else block
		}
	}

	return &ast.IfNode{
		Condition: condition,
		Body:      body,
		ElseIfs:   elseIfs,
		Else:      elseNode,
		Init:      init,
	}
}

// parseIncDecStmt parses an increment or decrement statement
func (p *Parser) parseIncDecStmt() ast.Node {
	ident := p.expect(ast.TokenIdentifier)
	op := p.current()
	if op.Type != ast.TokenPlusPlus && op.Type != ast.TokenMinusMinus {
		p.FailWithParseError(op, "Expected ++ or --")
	}
	p.advance() // Consume operator

	return ast.UnaryExpressionNode{
		Operator: op.Type,
		Operand: ast.VariableNode{
			Ident: ast.Ident{ID: ast.Identifier(ident.Value)},
		},
	}
}

// parseSendStmt parses a send statement (channel <- value)
func (p *Parser) parseSendStmt() ast.Node {
	ident := p.expect(ast.TokenIdentifier)
	p.expect(ast.TokenArrow)
	value := p.parseExpression()

	return ast.BinaryExpressionNode{
		Left: ast.VariableNode{
			Ident: ast.Ident{ID: ast.Identifier(ident.Value)},
		},
		Operator: ast.TokenArrow,
		Right:    value,
	}
}
