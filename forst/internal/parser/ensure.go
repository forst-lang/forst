package parser

import (
	"fmt"
	"forst/internal/ast"
	"strings"
)

// ensureMissingIsMessage explains that ensure requires `is` (bare Bool / `or` is not enough).
func ensureMissingIsMessage(subject string, found ast.Token) string {
	hint := fmt.Sprintf("for Bool use: ensure %s is True()", subject)
	switch found.Type {
	case ast.TokenOr:
		return fmt.Sprintf("ensure requires 'is' before 'or'; %s", hint)
	case ast.TokenGreater, ast.TokenLess, ast.TokenGreaterEqual, ast.TokenLessEqual,
		ast.TokenEquals, ast.TokenNotEquals:
		return fmt.Sprintf("ensure requires 'is' with a constraint (not a comparison); %s", hint)
	case ast.TokenLParen:
		return fmt.Sprintf("ensure subject must be an identifier (bind the call first); %s (found %s)", hint, found.Type)
	default:
		return fmt.Sprintf("ensure requires 'is' after the subject; %s (found %s)", hint, found.Type)
	}
}

func (p *Parser) parseEnsureBlock() *ast.EnsureBlockNode {
	body := []ast.Node{}

	// Ensure block is always optional
	if p.current().Type != ast.TokenLBrace {
		return nil
	}

	body = append(body, p.parseBlock()...)

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
			p.FailWithParseError(p.current(), "Expected variable after ensure !")
		}
		tok := p.current()
		variable = ast.VariableNode{
			Ident: ast.Ident{
				ID:   ast.Identifier(tok.Value),
				Span: ast.SpanFromToken(tok),
			},
		}
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
		// Parse the left side as a variable or field access (identifiers only — no call subjects).
		firstTok := p.expect(ast.TokenIdentifier)
		curIdent := ast.Identifier(firstTok.Value)
		lastTok := firstTok

		// Allow field access with dots
		for p.current().Type == ast.TokenDot {
			p.advance() // Consume dot
			nextTok := p.expect(ast.TokenIdentifier)
			curIdent = ast.Identifier(string(curIdent) + "." + nextTok.Value)
			lastTok = nextTok
		}

		subjectSpan := ast.SpanBetweenTokens(firstTok, lastTok)
		variable = ast.VariableNode{
			Ident: ast.Ident{ID: curIdent, Span: subjectSpan},
		}

		if p.current().Type != ast.TokenIs {
			p.FailWithParseError(p.current(), ensureMissingIsMessage(string(curIdent), p.current()))
		}
		p.expect(ast.TokenIs)
		if tok := p.current(); tok.Type == ast.TokenTrue || tok.Type == ast.TokenFalse {
			want := "True()"
			if tok.Type == ast.TokenFalse {
				want = "False()"
			}
			p.FailWithParseError(tok, fmt.Sprintf(
				"ensure predicate must be a constraint, not a boolean literal; use `ensure %s is %s`",
				curIdent, want,
			))
		}
		assertion = p.parseAssertionChain(false)

		// Try to set the base type from the current scope if not set (simple subject only).
		// For compound paths (e.g. ensure req.state is ValidBoard()), the subject's static type is
		// the field type (GameState), not the root variable's type (MoveRequest). Setting BaseType to
		// the latter breaks InferAssertionType and ensure-successor narrowing / hover.
		if assertion.BaseType == nil && p.context != nil && p.context.ScopeStack != nil {
			scope := p.context.ScopeStack.CurrentScope()
			if scope != nil {
				parts := strings.Split(string(curIdent), ".")
				baseIdent := parts[0]
				if typeNode, ok := scope.Variables[baseIdent]; ok && len(parts) == 1 {
					baseType := typeNode.Ident
					assertion.BaseType = &baseType
				}
			}
		}
	}

	block := p.parseEnsureBlock()

	if p.context.IsTypeGuard() {
		if p.current().Type == ast.TokenOr {
			p.FailWithParseError(p.current(), "Ensure statement not allowed in type guards")
		}
		return ast.EnsureNode{
			Variable:  variable,
			Assertion: assertion,
			Block:     block,
		}
	}

	if p.context.IsMainFunction() {
		if p.current().Type == ast.TokenOr {
			p.FailWithParseError(p.current(), "\"or\" block in ensure statements is not allowed in main function")
		}
	}

	// Only require 'or' clause if not in main function and not in a type guard context
	if !p.context.IsMainFunction() && p.current().Type == ast.TokenOr {
		p.expect(ast.TokenOr) // Expect `or`

		errorTok := p.expect(ast.TokenIdentifier)
		errorType := errorTok.Value
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
