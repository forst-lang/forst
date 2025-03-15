package parser

import (
	"forst/pkg/ast"
)

type BlockContext struct {
	AllowReturn bool
}

func (p *Parser) parseBlock(blockContext *BlockContext, context *Context) []ast.Node {
	body := []ast.Node{}

	p.expect(ast.TokenLBrace) // Expect `{`

	// Parse function body dynamically
	for p.current().Type != ast.TokenRBrace && p.current().Type != ast.TokenEOF {
		token := p.current()

		if token.Type == ast.TokenEnsure {
			body = append(body, p.parseEnsureStatement(context))
		} else if token.Type == ast.TokenReturn {
			if !blockContext.AllowReturn {
				panic(parseErrorWithValue(token, "Return statement not allowed in this context"))
			}
			body = append(body, p.parseReturnStatement(context))
		} else if token.Type == ast.TokenIdentifier {
			// Look ahead to see if this is a function call or assignment
			if p.peek().Type == ast.TokenLParen {
				// Function call
				body = append(body, p.parseExpression(context))
			} else if p.peek().Type == ast.TokenComma {
				// Multiple assignment
				firstIdent := p.expect(ast.TokenIdentifier)
				p.expect(ast.TokenComma)
				secondIdent := p.expect(ast.TokenIdentifier)

				// Expect assignment operator
				assignToken := p.current()
				if assignToken.Type != ast.TokenEquals && assignToken.Type != ast.TokenColonEquals {
					panic(parseErrorWithValue(assignToken, "Expected assignment or short assignment operator"))
				}
				p.advance()

				// Parse comma-separated expressions
				var exprs []ast.ExpressionNode
				for {
					exprs = append(exprs, p.parseExpression(context))
					if p.current().Type != ast.TokenComma {
						break
					}
					p.advance() // Skip comma
				}

				body = append(body, ast.AssignmentNode{
					Names:         []string{firstIdent.Value, secondIdent.Value},
					Values:        exprs,
					ExplicitTypes: []*ast.TypeNode{nil, nil},
					IsShort:       assignToken.Type == ast.TokenColonEquals,
				})
			} else {
				panic(parseErrorWithValue(token, "Expected function call or assignment after identifier"))
			}
		} else {
			panic(parseErrorWithValue(token, "Unexpected token in function body"))
		}
	}

	p.expect(ast.TokenRBrace) // Expect `}`

	return body
}
