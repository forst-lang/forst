package parser

import "forst/pkg/ast"

func (p *Parser) parseExpression() ast.ExpressionNode {
	return p.parseExpressionLevel(0)
}

func (p *Parser) parseExpressionLevel(level int) ast.ExpressionNode {
	if level > 100 {
		panic("Expression level too deep")
	}

	var expr ast.ExpressionNode

	// Handle unary not operator
	if p.current().Type == ast.TokenLogicalNot {
		p.advance() // Consume the not operator
		operand := p.parseExpressionLevel(level + 1)
		expr = ast.UnaryExpressionNode{
			Operator: ast.TokenLogicalNot,
			Operand:  operand,
		}
		return expr
	}

	// Handle parentheses
	if p.current().Type == ast.TokenLParen {
		p.advance() // Consume the left parenthesis
		expr = p.parseExpressionLevel(level + 1)
		p.expect(ast.TokenRParen) // Consume the right parenthesis
	} else {
		expr = p.parseLiteral() // parseLiteral should advance the token internally
	}

	// Handle binary operators
	for p.current().Type == ast.TokenPlus || p.current().Type == ast.TokenMinus ||
		p.current().Type == ast.TokenMultiply || p.current().Type == ast.TokenDivide {

		operator := p.current().Type
		p.advance() // Consume the operator

		right := p.parseExpressionLevel(level + 1)

		expr = ast.BinaryExpressionNode{
			Left:     expr,
			Operator: operator,
			Right:    right,
		}
	}

	return expr
}
