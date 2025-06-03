package parser

import (
	"forst/internal/ast"
)

func (p *Parser) parseEnsureBlock() *ast.EnsureBlockNode {
	body := []ast.Node{}

	// Ensure block is always optional
	if p.current().Type != ast.TokenLBrace {
		return nil
	}

	body = append(body, p.parseBlock(&BlockContext{AllowReturn: false})...)

	return &ast.EnsureBlockNode{Body: body}
}

func (p *Parser) parseEnsureStatement() ast.EnsureNode {
	p.advance() // Move past `ensure`

	var variable ast.VariableNode
	var assertion ast.AssertionNode

	// Handle special case for negated variable check
	if p.current().Type == ast.TokenLogicalNot && p.peek().Type == ast.TokenIdentifier {
		p.advance() // Move past !
		if p.peek().Type == ast.TokenLParen {
			panic(parseErrorWithValue(p.current(), "Expected variable after ensure !"))
		}
		variable = ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(p.current().Value)}}
		p.advance() // Move past variable
		// Create implicit Nil() assertion
		errorType := ast.TypeError
		assertion = ast.AssertionNode{
			BaseType: &errorType,
			Constraints: []ast.ConstraintNode{
				{
					Name: "Nil",
					Args: []ast.ConstraintArgumentNode{},
				},
			},
		}
	} else {
		// Parse the left side as a variable or field access
		ident := p.expect(ast.TokenIdentifier)
		curIdent := ast.Identifier(ident.Value)

		// Allow field access with dots
		for p.current().Type == ast.TokenDot {
			p.advance() // Consume dot
			nextIdent := p.expect(ast.TokenIdentifier)
			curIdent = ast.Identifier(string(curIdent) + "." + nextIdent.Value)
		}

		variable = ast.VariableNode{
			Ident: ast.Ident{ID: curIdent},
		}

		p.expect(ast.TokenIs)
		assertion = p.parseAssertionChain(false)
	}

	block := p.parseEnsureBlock()

	if !p.context.IsMainFunction() || p.current().Type == ast.TokenOr {
		p.expect(ast.TokenOr) // Expect `or`

		errorType := p.expect(ast.TokenIdentifier).Value
		var err ast.EnsureErrorNode
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
			err = ast.EnsureErrorCall{ErrorType: errorType, ErrorArgs: args}
		} else {
			err = ast.EnsureErrorVar(errorType)
		}
		return ast.EnsureNode{
			Variable:  variable,
			Assertion: assertion,
			Block:     block,
			Error:     &err,
		}
	}

	return ast.EnsureNode{
		Variable:  variable,
		Assertion: assertion,
		Block:     block,
	}
}
