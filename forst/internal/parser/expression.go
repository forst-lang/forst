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
		if p.current().Type == ast.TokenEllipsis {
			p.advance()
			arg = ast.SpreadExpressionNode{Expr: arg}
		}
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
		// Single `=` is assignment at statement level only, not a binary expression operator.
		if tok == ast.TokenEquals && p.current().Value == "=" {
			break
		}
		// `* *` begins a dereference chain / assign target, not multiplication.
		if tok == ast.TokenStar && p.peek().Type == ast.TokenStar {
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
				right := p.parseShapeLiteral(ShapeLiteralOpts{})
				lhs = ast.BinaryExpressionNode{
					Left:     lhs,
					Operator: operator,
					Right:    right,
				}
			} else if p.current().Type == ast.TokenIdentifier && p.current().Value == "Shape" {
				p.advance()
				p.expect(ast.TokenLParen)
				if p.current().Type == ast.TokenLBrace {
					right := p.parseShapeLiteral(ShapeLiteralOpts{})
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

// parseIndexSuffixChain parses zero or more `[expr]` / `[low:high]` suffixes (slice/array indexing).
func (p *Parser) parseIndexSuffixChain(base ast.ExpressionNode, _ int) ast.ExpressionNode {
	for p.current().Type == ast.TokenLBracket {
		p.advance()
		if p.current().Type == ast.TokenColon {
			p.advance()
			var high ast.ExpressionNode
			if p.current().Type != ast.TokenRBracket {
				high = p.parseExpression()
			}
			p.expect(ast.TokenRBracket)
			base = ast.SliceExpressionNode{Target: base, High: high}
			continue
		}
		first := p.parseExpression()
		if p.current().Type == ast.TokenColon {
			p.advance()
			var high ast.ExpressionNode
			if p.current().Type != ast.TokenRBracket {
				high = p.parseExpression()
			}
			p.expect(ast.TokenRBracket)
			base = ast.SliceExpressionNode{Target: base, Low: first, High: high}
			continue
		}
		p.expect(ast.TokenRBracket)
		base = ast.IndexExpressionNode{
			Target: base,
			Index:  first,
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
	case p.current().Type == ast.TokenMinus:
		p.advance()
		base = ast.UnaryExpressionNode{
			Operator: ast.TokenMinus,
			Operand:  p.parseUnaryOrPrimary(depth + 1),
		}
	case p.current().Type == ast.TokenStar:
		p.advance()
		operand := p.parseUnaryOrPrimary(depth + 1)
		vn, ok := operand.(ast.ValueNode)
		if !ok {
			p.FailWithParseError(p.current(), "dereference operand must be a value expression")
		}
		base = ast.DereferenceNode{Value: vn}
	case p.current().Type == ast.TokenLParen:
		p.advance()
		inner := p.parseExpr(0, depth+1)
		p.expect(ast.TokenRParen)
		base = inner
	case p.current().Type == ast.TokenLBracket:
		if conv, ok := p.tryParseSliceConversion(); ok {
			base = conv
		} else {
			base = p.parseValue()
		}
	case p.current().Type == ast.TokenIdentifier:
		base = p.parseIdentifierPrimary()
	case p.current().Type == ast.TokenInt, p.current().Type == ast.TokenString,
		p.current().Type == ast.TokenBool, p.current().Type == ast.TokenFloat:
		typeTok := p.current()
		p.advance()
		if p.current().Type == ast.TokenLParen {
			p.FailWithParseError(typeTok, typeTok.Value+" is a type, not a conversion; use Go stdlib (e.g. strconv) or string() for formatting")
		}
		p.FailWithParseError(typeTok, typeTok.Value+" is a type name, not an expression")
	case p.current().Type == ast.TokenNil:
		p.advance()
		base = ast.NilLiteralNode{}
	default:
		base = p.parseValue()
	}

	return p.parsePostfixSuffixChain(base, depth)
}

// parsePostfixSuffixChain parses [index], .method(), and .field suffixes.
func (p *Parser) parsePostfixSuffixChain(base ast.ExpressionNode, depth int) ast.ExpressionNode {
	base = p.parseIndexSuffixChain(base, depth)
	for p.current().Type == ast.TokenDot {
		p.advance()
		memberTok := p.parseSelectorName()
		if p.current().Type == ast.TokenLParen {
			lparen := p.current()
			p.advance()
			args, argSpans := p.parseCallArguments()
			rparen := p.expect(ast.TokenRParen)
			base = ast.MethodCallNode{
				Receiver:  base,
				Method:    ast.Ident{ID: ast.Identifier(memberTok.Value), Span: ast.SpanFromToken(memberTok)},
				Arguments: args,
				CallSpan:  ast.SpanBetweenTokens(lparen, rparen),
				ArgSpans:  argSpans,
			}
		} else {
			base = ast.FieldAccessNode{
				Target: base,
				Field:  ast.Ident{ID: ast.Identifier(memberTok.Value), Span: ast.SpanFromToken(memberTok)},
			}
		}
	}
	return base
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
		if ident.ID == "Ok" || ident.ID == "Err" {
			if len(args) != 1 {
				p.FailWithParseError(lparen, string(ident.ID)+" expects exactly one argument")
			}
			_ = argSpans
			_ = rparen
			if ident.ID == "Ok" {
				return ast.OkExprNode{Value: args[0]}
			}
			return ast.ErrExprNode{Value: args[0]}
		}
		return ast.FunctionCallNode{
			Function:  ident,
			Arguments: args,
			CallSpan:  ast.SpanBetweenTokens(lparen, rparen),
			ArgSpans:  argSpans,
		}
	}
	if p.current().Type == ast.TokenLBrace && isShapeLiteralTypePrefix(string(ident.ID)) {
		typeIdent := ast.TypeIdent(string(ident.ID))
		return p.parseShapeLiteral(ShapeLiteralOpts{BaseType: &typeIdent})
	}
	return ast.VariableNode{
		Ident: ident,
	}
}

// tryParseSliceConversion parses Go-style []T(expr) slice conversions such as []byte(s).
func (p *Parser) tryParseSliceConversion() (ast.ExpressionNode, bool) {
	if p.peek().Type != ast.TokenRBracket {
		return nil, false
	}
	saved := p.currentIndex
	lbrack := p.current()
	p.advance() // [
	rbrack := p.expect(ast.TokenRBracket)
	elemType := p.parseOptionalArrayElementTypeSuffix(rbrack)
	if elemType.Ident == ast.TypeImplicit || p.current().Type != ast.TokenLParen {
		p.currentIndex = saved
		return nil, false
	}
	if isBuiltinTypeKeywordIdent(elemType.Ident) {
		p.FailWithParseError(rbrack, "[]"+string(elemType.Ident)+" is not a slice conversion; use Go stdlib or []byte(s) for string-to-bytes")
	}
	convName := ast.Identifier("[]" + string(elemType.Ident))
	lparen := p.current()
	p.advance()
	args, argSpans := p.parseCallArguments()
	rparen := p.expect(ast.TokenRParen)
	return ast.FunctionCallNode{
		Function:  ast.Ident{ID: convName, Span: ast.SpanBetweenTokens(lbrack, rparen)},
		Arguments: args,
		CallSpan:  ast.SpanBetweenTokens(lparen, rparen),
		ArgSpans:  argSpans,
	}, true
}

func isBuiltinTypeKeywordIdent(ident ast.TypeIdent) bool {
	switch ident {
	case ast.TypeInt, ast.TypeString, ast.TypeFloat, ast.TypeBool:
		return true
	default:
		return false
	}
}
