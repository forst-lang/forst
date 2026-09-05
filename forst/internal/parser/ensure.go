package parser

import (
	"fmt"
	"forst/internal/ast"
	"strings"
)

// ensureMissingIsReport explains that ensure requires `is` (bare Bool / `or` is not enough).
func ensureMissingIsReport(subject string, found ast.Token) (code, title, problem, help string) {
	help = fmt.Sprintf("for Bool values write:\n\n    ensure %s is True()", subject)
	switch found.Type {
	case ast.TokenOr:
		return "ensure-missing-is", "ensure needs `is` before `or`",
			"After the subject, Forst expects `is` plus a constraint — not a bare `or`.",
			help
	case ast.TokenGreater, ast.TokenLess, ast.TokenGreaterEqual, ast.TokenLessEqual,
		ast.TokenEquals, ast.TokenNotEquals, ast.TokenLogicalOr, ast.TokenLogicalAnd:
		return "ensure-missing-is", "ensure needs `is`, not a comparison",
			"ensure requires `is` with a constraint — not a boolean comparison.",
			help
	case ast.TokenLParen:
		return "ensure-missing-is", "ensure subject must be a name",
			"Bind the call to a variable first, then ensure on that name.",
			help
	default:
		return "ensure-missing-is", "ensure needs `is`",
			fmt.Sprintf("After the subject, Forst expects `is` plus a constraint (found %s).", found.Type),
			help
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
			p.FailWithReport(p.current(), "ensure-negation-subject", "ensure ! needs a variable",
				"After `ensure !`, Forst expects a variable name, not a call.",
				"write `ensure !flag is True()`")
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
			p.FailWithReport(p.current(), "refinement-non-place-subject", "ensure subject must be a place",
				"ensure subject must be an identifier or field path, not a literal.",
				"bind the value to a variable, then ensure on that name")
		case ast.TokenLParen:
			p.FailWithReport(p.current(), "refinement-non-place-subject", "ensure subject must be a place",
				"ensure subject must be an identifier or field path.",
				"bind the expression to a variable, then ensure on that name")
		}

		if p.current().Type != ast.TokenIdentifier {
			code, title, problem, help := ensureMissingIsReport("?", p.current())
			p.FailWithReport(p.current(), code, title, problem, help)
		}

		// Parse the left side as a variable or field access (identifiers only — no call subjects).
		firstTok := p.expect(ast.TokenIdentifier)
		// Call subject: ident(
		if p.current().Type == ast.TokenLParen {
			p.FailWithReport(firstTok, "refinement-non-place-subject", "ensure subject must be a place",
				"ensure subject must be an identifier (bind the call first).",
				"assign the call result to a variable, then ensure on that name")
		}
		curIdent := ast.Identifier(firstTok.Value)
		lastTok := firstTok

		// Allow field access with dots (but not method calls)
		for p.current().Type == ast.TokenDot {
			p.advance() // Consume dot
			nextTok := p.expect(ast.TokenIdentifier)
			if p.current().Type == ast.TokenLParen {
				p.FailWithReport(nextTok, "refinement-non-place-subject", "ensure subject must be a place",
					"ensure subject must be an identifier or field path, not a call.",
					"bind the call result to a variable, then ensure on that name")
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
				p.FailWithReport(tok, "refinement-non-place-subject", "ensure subject must be a place",
					fmt.Sprintf("ensure subject must be a place, not an expression (%s).", tok.Type),
					"bind the expression to a variable, then ensure on that name")
			}
			code, title, problem, help := ensureMissingIsReport(string(curIdent), tok)
			p.FailWithReport(tok, code, title, problem, help)
		}
		p.expect(ast.TokenIs)
		if tok := p.current(); tok.Type == ast.TokenTrue || tok.Type == ast.TokenFalse {
			want := "True()"
			if tok.Type == ast.TokenFalse {
				want = "False()"
			}
			p.FailWithReport(tok, "ensure-boolean-literal", "ensure predicate must be a constraint",
				"ensure predicate must be a constraint, not a boolean literal.",
				fmt.Sprintf("use `ensure %s is %s`", curIdent, want))
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

	// Failure handling: `else <Error()|errVar>` or `else { … }`
	if p.current().Type == ast.TokenElse {
		elseTok := p.current()
		// `else if` belongs to surrounding control flow
		if p.peek().Type != ast.TokenIf {
			p.advance() // consume else
			if p.current().Type == ast.TokenLBrace {
				if inGuard {
					p.FailWithReport(elseTok, "refinement-failure-block-in-guard", "failure blocks are not allowed inside type guards",
						"Typed failure blocks (`else { … }`) cannot appear inside type guards.",
						"use a typed `else Error()` or move the ensure outside the guard")
				}
				block = p.parseEnsureBlock()
			} else {
				if inGuard {
					p.FailWithReport(elseTok, "refinement-else-in-guard", "typed `else` is not allowed inside type guards",
						"Typed failure (`else Error()`) cannot appear inside type guards.",
						"move the ensure outside the guard or use a failure block only in ordinary functions")
				}
				if inMain {
					p.FailWithReport(elseTok, "refinement-else-in-main", "typed failure is not allowed in main",
						`"else" typed failure in ensure statements is not allowed in main function.`,
						"use a failure block `else { … }` in main, or move typed failure to another function")
				}
				errNode = p.parseEnsureError()
				if p.current().Type == ast.TokenLBrace {
					p.FailWithReport(p.current(), "refinement-else-and-block", "cannot combine typed `else` and failure block",
						"use either `else <error>` or a failure block `else { … }`, not both.",
						"pick one failure form per ensure statement")
				}
			}
		}
	} else if p.current().Type == ast.TokenLBrace {
		if inGuard {
			p.FailWithReport(p.current(), "refinement-failure-block-in-guard", "failure blocks are not allowed inside type guards",
				"Typed failure blocks (`else { … }`) cannot appear inside type guards.",
				"use a typed `else Error()` or move the ensure outside the guard")
		}
		p.FailWithReport(p.current(), "refinement-bare-ensure-block", "ensure failure block requires `else`",
			"ensure failure block requires 'else'; write: ensure … else { … }",
			"prefix the block with `else`")
	}

	// `else` after block
	if p.current().Type == ast.TokenElse && block != nil {
		p.FailWithReport(p.current(), "refinement-else-and-block", "cannot combine typed `else` and failure block",
			"use either `else <error>` or a failure block `else { … }`, not both.",
			"pick one failure form per ensure statement")
	}

	// Legacy `or` as typed failure: if somehow still present after a complete target
	// without having been consumed as Join (e.g. orphaned), suggest else.
	if p.current().Type == ast.TokenOr && errNode == nil && block == nil {
		p.FailWithReport(p.current(), "refinement-legacy-failure-or", "typed failure uses `else`, not `or`",
			"typed failure uses `else`, not `or`; `or` joins assertion alternatives.",
			"write `ensure x is Foo() else MyError()` instead of `… or MyError()`")
	}

	return ast.EnsureNode{
		Variable:  variable,
		Target:    target,
		Assertion: assertion,
		Block:     block,
		Error:     errNode,
	}
}
