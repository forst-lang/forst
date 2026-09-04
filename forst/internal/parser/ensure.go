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
		ast.TokenEquals, ast.TokenNotEquals, ast.TokenLogicalOr, ast.TokenLogicalAnd:
		return fmt.Sprintf("refinement-boolean-ensure: ensure requires 'is' with a constraint (not a comparison); %s", hint)
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

// parseEnsureError parses the typed failure after `else` (Error() call or error variable).
func (p *Parser) parseEnsureError() *ast.EnsureErrorNode {
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
	return &err
}

func (p *Parser) parseEnsureStatement() ast.EnsureNode {
	p.advance() // Move past `ensure`

	var variable ast.VariableNode
	var assertion ast.AssertionNode
	var target ast.RefinementTarget

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
		target = ast.AssertionTarget{Chains: []ast.AssertionNode{assertion}}
	} else {
		// Reject non-place subjects early (calls, literals, arithmetic).
		switch p.current().Type {
		case ast.TokenStringLiteral, ast.TokenIntLiteral, ast.TokenFloatLiteral,
			ast.TokenRuneLiteral, ast.TokenTrue, ast.TokenFalse, ast.TokenNil:
			p.FailWithParseError(p.current(),
				"refinement-non-place-subject: ensure subject must be an identifier or field path, not a literal")
		case ast.TokenLParen:
			p.FailWithParseError(p.current(),
				"refinement-non-place-subject: ensure subject must be an identifier or field path")
		}

		if p.current().Type != ast.TokenIdentifier {
			p.FailWithParseError(p.current(), ensureMissingIsMessage("?", p.current()))
		}

		// Parse the left side as a variable or field access (identifiers only — no call subjects).
		firstTok := p.expect(ast.TokenIdentifier)
		// Call subject: ident(
		if p.current().Type == ast.TokenLParen {
			p.FailWithParseError(firstTok,
				"refinement-non-place-subject: ensure subject must be an identifier (bind the call first)")
		}
		curIdent := ast.Identifier(firstTok.Value)
		lastTok := firstTok

		// Allow field access with dots (but not method calls)
		for p.current().Type == ast.TokenDot {
			p.advance() // Consume dot
			nextTok := p.expect(ast.TokenIdentifier)
			if p.current().Type == ast.TokenLParen {
				p.FailWithParseError(nextTok,
					"refinement-non-place-subject: ensure subject must be an identifier or field path, not a call")
			}
			curIdent = ast.Identifier(string(curIdent) + "." + nextTok.Value)
			lastTok = nextTok
		}

		// Reject arithmetic / comparison subjects: `a + b is …` never starts with two idents.
		// `ensure a + b` hits `+` before `is`.
		if p.current().Type != ast.TokenIs {
			tok := p.current()
			if tok.Type.IsArithmeticBinaryOperator() || tok.Type.IsComparisonBinaryOperator() ||
				tok.Type == ast.TokenLogicalOr || tok.Type == ast.TokenLogicalAnd {
				p.FailWithParseError(tok,
					fmt.Sprintf("refinement-non-place-subject: ensure subject must be a place, not an expression (%s)", tok.Type))
			}
			p.FailWithParseError(tok, ensureMissingIsMessage(string(curIdent), tok))
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

		subjectSpan := ast.SpanBetweenTokens(firstTok, lastTok)
		variable = ast.VariableNode{
			Ident: ast.Ident{ID: curIdent, Span: subjectSpan},
		}

		target, assertion = p.parseRefinementTarget()

		// Try to set the base type from the current scope if not set (simple subject only).
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

	inGuard := p.context != nil && p.context.IsTypeGuard()
	inMain := p.context != nil && p.context.IsMainFunction()

	var errNode *ast.EnsureErrorNode
	var block *ast.EnsureBlockNode

	// Optional typed failure: `else <Error()|errVar>`
	if p.current().Type == ast.TokenElse {
		elseTok := p.current()
		// `else if` / `else {` belong to surrounding if — but after ensure, `else {` is invalid
		// (failure blocks use bare `{`; typed else takes Error()/var).
		if p.peek().Type == ast.TokenIf {
			// Not an ensure else; leave for outer parse (shouldn't appear mid-ensure).
		} else {
			p.advance() // consume else
			if inGuard {
				p.FailWithParseError(elseTok,
					"refinement-else-in-guard: typed `else` is not allowed inside type guards")
			}
			if inMain {
				p.FailWithParseError(elseTok,
					`"else" typed failure in ensure statements is not allowed in main function`)
			}
			if p.current().Type == ast.TokenLBrace {
				p.FailWithParseError(elseTok,
					"refinement-else-and-block: use either `else <error>` or a failure block `{ … }`, not both")
			}
			errNode = p.parseEnsureError()
		}
	}

	// Optional failure block `{ … }` (XOR with else)
	if p.current().Type == ast.TokenLBrace {
		if errNode != nil {
			p.FailWithParseError(p.current(),
				"refinement-else-and-block: use either `else <error>` or a failure block `{ … }`, not both")
		}
		if inGuard {
			p.FailWithParseError(p.current(),
				"refinement-failure-block-in-guard: failure blocks are not allowed inside type guards")
		}
		block = p.parseEnsureBlock()
	}

	// `else` after block
	if p.current().Type == ast.TokenElse && block != nil {
		p.FailWithParseError(p.current(),
			"refinement-else-and-block: use either `else <error>` or a failure block `{ … }`, not both")
	}

	// Legacy `or` as typed failure: if somehow still present after a complete target
	// without having been consumed as Join (e.g. orphaned), suggest else.
	if p.current().Type == ast.TokenOr && errNode == nil && block == nil {
		p.FailWithParseError(p.current(),
			"refinement-legacy-failure-or: typed failure uses `else`, not `or`; `or` joins assertion alternatives")
	}

	return ast.EnsureNode{
		Variable:  variable,
		Target:    target,
		Assertion: assertion,
		Block:     block,
		Error:     errNode,
	}
}
