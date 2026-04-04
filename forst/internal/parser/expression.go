package parser

import (
	"fmt"

	"forst/internal/ast"
)

// parseCallArguments parses comma-separated expressions until ')'. Caller must have consumed '('.
func (p *Parser) parseCallArguments() (args []ast.ExpressionNode, argSpans []ast.SourceSpan) {
	for p.current().Type != ast.TokenRParen {
		startTok := p.current()
		arg := p.parseExpression()
		endTok := p.tokens[p.currentIndex-1]
		args = append(args, arg)
		argSpans = append(argSpans, ast.SpanBetweenTokens(startTok, endTok))
		if p.current().Type == ast.TokenComma {
			p.advance()
		}
	}
	return args, argSpans
}

// MaxExpressionDepth is the maximum depth of nested expressions
const MaxExpressionDepth = 20

// parseExpression parses an expression at the current level
func (p *Parser) parseExpression() ast.ExpressionNode {
	return p.parseExpressionLevel(0)
}

func (p *Parser) parseExpressionLevel(level int) ast.ExpressionNode {
	if level > MaxExpressionDepth {
		p.FailWithParseError(p.current(), fmt.Sprintf("Expression level too deep - maximum nesting depth is %d", MaxExpressionDepth))
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

		// If we have an identifier followed by a left parenthesis, this is a function call
		if p.current().Type == ast.TokenIdentifier {
			identTok := p.expect(ast.TokenIdentifier)
			if p.current().Type == ast.TokenLParen {
				lparen := p.current()
				p.advance() // Consume left paren
				args, argSpans := p.parseCallArguments()
				rparen := p.expect(ast.TokenRParen)
				expr = ast.FunctionCallNode{
					Function:  ast.Ident{ID: ast.Identifier(identTok.Value), Span: ast.SpanFromToken(identTok)},
					Arguments: args,
					CallSpan:  ast.SpanBetweenTokens(lparen, rparen),
					ArgSpans:  argSpans,
				}
			}
		} else {
			p.expect(ast.TokenRParen) // Consume the right parenthesis
		}
	} else if p.current().Type == ast.TokenIdentifier {
		// Parse the identifier, allowing for dot chaining
		ident := p.parseIdentifier()
		// If we hit a left paren, this is a function call
		if p.current().Type == ast.TokenLParen {
			lparen := p.current()
			p.advance() // Consume left paren
			args, argSpans := p.parseCallArguments()
			rparen := p.expect(ast.TokenRParen)

			expr = ast.FunctionCallNode{
				Function:  ident,
				Arguments: args,
				CallSpan:  ast.SpanBetweenTokens(lparen, rparen),
				ArgSpans:  argSpans,
			}
		} else if p.current().Type == ast.TokenLBrace {
			typeIdent := ast.TypeIdent(string(ident.ID))
			// Parse the shape literal with the base type
			return p.parseShapeLiteral(&typeIdent, false)
		} else {
			// Otherwise treat as a variable
			expr = ast.VariableNode{
				Ident: ident,
			}
		}
	} else if p.current().Type == ast.TokenNil {
		p.advance()
		return ast.NilLiteralNode{}
	} else {
		expr = p.parseValue() // parseValue should advance the token internally
	}

	// Handle binary operators
	for p.current().Type.IsBinaryOperator() {
		operator := p.current().Type
		p.advance() // Consume the operator

		// Special handling for 'is' operator
		if operator == ast.TokenIs {
			// Check if the right-hand side is a shape literal or Shape(...) call
			if p.current().Type == ast.TokenLBrace {
				right := p.parseShapeLiteral(nil, false) // Parse the shape literal directly
				expr = ast.BinaryExpressionNode{
					Left:     expr,
					Operator: operator,
					Right:    right,
				}
			} else if p.current().Type == ast.TokenIdentifier && p.current().Value == "Shape" {
				p.advance() // Consume 'Shape'
				p.expect(ast.TokenLParen)
				if p.current().Type == ast.TokenLBrace {
					right := p.parseShapeLiteral(nil, false) // Parse the shape literal directly
					p.expect(ast.TokenRParen)
					expr = ast.BinaryExpressionNode{
						Left:     expr,
						Operator: operator,
						Right:    right,
					}
				} else {
					right := p.parseExpressionLevel(level + 1)
					p.expect(ast.TokenRParen)
					expr = ast.BinaryExpressionNode{
						Left:     expr,
						Operator: operator,
						Right:    right,
					}
				}
			} else if p.current().Type == ast.TokenIdentifier {
				// Parse assertion node for the right-hand side
				assertion := p.parseAssertionChain(false)
				expr = ast.BinaryExpressionNode{
					Left:     expr,
					Operator: operator,
					Right:    assertion,
				}
			} else {
				right := p.parseExpressionLevel(level + 1)
				expr = ast.BinaryExpressionNode{
					Left:     expr,
					Operator: operator,
					Right:    right,
				}
			}
		} else {
			right := p.parseExpressionLevel(level + 1)
			expr = ast.BinaryExpressionNode{
				Left:     expr,
				Operator: operator,
				Right:    right,
			}
		}
	}

	return expr
}
