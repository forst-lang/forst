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

// binaryPrecedence returns operator precedence for Pratt parsing. Higher number = tighter binding
// (binds before operators with lower numbers). Must match Go: && tighter than ||, comparisons
// tighter than &&, etc.
func binaryPrecedence(op ast.TokenIdent) int {
	switch op {
	case ast.TokenLogicalOr:
		return 10
	case ast.TokenLogicalAnd:
		return 20
	case ast.TokenEquals, ast.TokenNotEquals, ast.TokenGreater, ast.TokenLess,
		ast.TokenGreaterEqual, ast.TokenLessEqual, ast.TokenIs:
		return 30
	case ast.TokenPlus, ast.TokenMinus:
		return 40
	case ast.TokenStar, ast.TokenDivide, ast.TokenModulo:
		return 50
	default:
		return 0
	}
}

// parseExpression parses an expression with correct binary operator precedence (e.g. comparisons
// bind tighter than && so `a != x && a != y` is `(a != x) && (a != y)`).
func (p *Parser) parseExpression() ast.ExpressionNode {
	return p.parseExpr(0, 0)
}

// parseExpr implements Pratt parsing: left-associative binary operators with minPrec threshold.
func (p *Parser) parseExpr(minPrec int, depth int) ast.ExpressionNode {
	if depth > MaxExpressionDepth {
		p.FailWithParseError(p.current(), fmt.Sprintf("Expression nesting too deep - maximum depth is %d", MaxExpressionDepth))
	}

	lhs := p.parseUnaryOrPrimary(depth)
	for {
		tok := p.current().Type
		if !tok.IsBinaryOperator() {
			break
		}
		prec := binaryPrecedence(tok)
		if prec < minPrec {
			break
		}
		operator := tok
		p.advance()

		if operator == ast.TokenIs {
			if p.current().Type == ast.TokenLBrace {
				right := p.parseShapeLiteral(nil, false)
				lhs = ast.BinaryExpressionNode{
					Left:     lhs,
					Operator: operator,
					Right:    right,
				}
			} else if p.current().Type == ast.TokenIdentifier && p.current().Value == "Shape" {
				p.advance()
				p.expect(ast.TokenLParen)
				if p.current().Type == ast.TokenLBrace {
					right := p.parseShapeLiteral(nil, false)
					p.expect(ast.TokenRParen)
					lhs = ast.BinaryExpressionNode{
						Left:     lhs,
						Operator: operator,
						Right:    right,
					}
				} else {
					right := p.parseExpr(0, depth+1)
					p.expect(ast.TokenRParen)
					lhs = ast.BinaryExpressionNode{
						Left:     lhs,
						Operator: operator,
						Right:    right,
					}
				}
			} else if p.current().Type == ast.TokenIdentifier {
				assertion := p.parseAssertionChain(false)
				lhs = ast.BinaryExpressionNode{
					Left:     lhs,
					Operator: operator,
					Right:    assertion,
				}
			} else {
				right := p.parseExpr(0, depth+1)
				lhs = ast.BinaryExpressionNode{
					Left:     lhs,
					Operator: operator,
					Right:    right,
				}
			}
			continue
		}

		rhs := p.parseExpr(prec+1, depth+1)
		lhs = ast.BinaryExpressionNode{
			Left:     lhs,
			Operator: operator,
			Right:    rhs,
		}
	}
	return lhs
}

// parseIndexSuffixChain parses zero or more `[expr]` suffixes (slice/array indexing).
func (p *Parser) parseIndexSuffixChain(base ast.ExpressionNode, depth int) ast.ExpressionNode {
	for p.current().Type == ast.TokenLBracket {
		p.advance()
		idx := p.parseExpression()
		p.expect(ast.TokenRBracket)
		base = ast.IndexExpressionNode{
			Target: base,
			Index:  idx,
		}
	}
	return base
}

func (p *Parser) parseUnaryOrPrimary(depth int) ast.ExpressionNode {
	if depth > MaxExpressionDepth {
		p.FailWithParseError(p.current(), fmt.Sprintf("Expression nesting too deep - maximum depth is %d", MaxExpressionDepth))
	}

	var base ast.ExpressionNode

	switch {
	case p.current().Type == ast.TokenLogicalNot:
		p.advance()
		base = ast.UnaryExpressionNode{
			Operator: ast.TokenLogicalNot,
			Operand:  p.parseUnaryOrPrimary(depth + 1),
		}
	case p.current().Type == ast.TokenLParen:
		p.advance()
		inner := p.parseExpr(0, depth+1)
		p.expect(ast.TokenRParen)
		base = inner
	case p.current().Type == ast.TokenIdentifier:
		base = p.parseIdentifierPrimary()
	case p.current().Type == ast.TokenNil:
		p.advance()
		base = ast.NilLiteralNode{}
	default:
		base = p.parseValue()
	}

	return p.parseIndexSuffixChain(base, depth)
}

// parseIdentifierPrimary parses call, typed shape literal, or variable; caller must be positioned
// on TokenIdentifier (it calls parseIdentifier).
func (p *Parser) parseIdentifierPrimary() ast.ExpressionNode {
	ident := p.parseIdentifier()
	if p.current().Type == ast.TokenLParen {
		lparen := p.current()
		p.advance()
		args, argSpans := p.parseCallArguments()
		rparen := p.expect(ast.TokenRParen)
		return ast.FunctionCallNode{
			Function:  ident,
			Arguments: args,
			CallSpan:  ast.SpanBetweenTokens(lparen, rparen),
			ArgSpans:  argSpans,
		}
	}
	if p.current().Type == ast.TokenLBrace && isShapeLiteralTypePrefix(string(ident.ID)) {
		typeIdent := ast.TypeIdent(string(ident.ID))
		return p.parseShapeLiteral(&typeIdent, false)
	}
	return ast.VariableNode{
		Ident: ident,
	}
}
