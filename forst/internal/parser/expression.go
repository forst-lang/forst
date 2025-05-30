package parser

import (
	"fmt"
	"forst/internal/ast"
)

const MAX_EXPRESSION_DEPTH = 20

func (p *Parser) parseExpression() ast.ExpressionNode {
	return p.parseExpressionLevel(0)
}

func (p *Parser) parseExpressionLevel(level int) ast.ExpressionNode {
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
		// Check for function call
		p.expect(ast.TokenRParen) // Consume the right parenthesis
	} else if p.current().Type == ast.TokenIdentifier {
		// Parse the identifier, allowing for dot chaining
		ident := p.expect(ast.TokenIdentifier)
		var curIdent ast.Identifier = ast.Identifier(ident.Value)

		// Keep chaining identifiers with dots until we hit something else
		for p.current().Type == ast.TokenDot {
			p.advance() // Consume dot
			nextIdent := p.expect(ast.TokenIdentifier)
			curIdent = ast.Identifier(string(curIdent) + "." + nextIdent.Value)
		}

		// If we hit a left paren, this is a function call
		if p.current().Type == ast.TokenLParen {
			p.advance() // Consume left paren

			var args []ast.ExpressionNode
			for p.current().Type != ast.TokenRParen {
				args = append(args, p.parseExpression())
				if p.current().Type == ast.TokenComma {
					p.advance()
				}
			}
			p.expect(ast.TokenRParen)

			expr = ast.FunctionCallNode{
				Function:  ast.Ident{Id: curIdent},
				Arguments: args,
			}
		} else {
			// Otherwise treat as a variable
			expr = ast.VariableNode{
				Ident: ast.Ident{Id: curIdent},
			}
		}
	} else {
		expr = p.parseValue() // parseValue should advance the token internally
	}

	// Handle binary operators
	for p.current().Type.IsBinaryOperator() {
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
