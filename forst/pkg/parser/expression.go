package parser

import (
	"fmt"
	"forst/pkg/ast"
)

const MAX_EXPRESSION_DEPTH = 20

func (p *Parser) parseExpression(context *Context) ast.ExpressionNode {
	return p.parseExpressionLevel(0, context)
}

func (p *Parser) parseExpressionLevel(level int, context *Context) ast.ExpressionNode {
	if level > MAX_EXPRESSION_DEPTH {
		panic(parseErrorWithValue(
			p.current(),
			fmt.Sprintf("Expression level too deep - maximum nesting depth is %d", MAX_EXPRESSION_DEPTH),
		))
	}

	var expr ast.ExpressionNode

	// Handle unary not operator
	if p.current().Type == ast.TokenLogicalNot {
		p.advance() // Consume the not operator
		operand := p.parseExpressionLevel(level+1, context)
		expr = ast.UnaryExpressionNode{
			Operator: ast.TokenLogicalNot,
			Operand:  operand,
		}
		return expr
	}

	// Handle parentheses
	if p.current().Type == ast.TokenLParen {
		p.advance() // Consume the left parenthesis
		expr = p.parseExpressionLevel(level+1, context)
		// Check for function call
		p.expect(ast.TokenRParen) // Consume the right parenthesis
	} else if p.current().Type == ast.TokenIdentifier && p.peek().Type == ast.TokenLParen {
		ident := p.expect(ast.TokenIdentifier)
		p.expect(ast.TokenLParen)

		var args []ast.ExpressionNode
		for p.current().Type != ast.TokenRParen {
			args = append(args, p.parseExpression(context))
			if p.current().Type == ast.TokenComma {
				p.advance()
			}
		}
		p.expect(ast.TokenRParen)

		expr = ast.FunctionCallNode{
			Function:  ast.Ident{Id: ast.Identifier(ident.Value)},
			Arguments: args,
		}
		return expr
	} else {
		expr = p.parseValue() // parseValue should advance the token internally
	}

	// Handle binary operators
	for p.current().Type.IsBinaryOperator() {
		operator := p.current().Type
		p.advance() // Consume the operator

		right := p.parseExpressionLevel(level+1, context)

		expr = ast.BinaryExpressionNode{
			Left:     expr,
			Operator: operator,
			Right:    right,
		}
	}

	return expr
}
