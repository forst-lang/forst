package parser

import (
	"fmt"
	"forst/internal/ast"
)

func isPossibleConstraintIdentifier(token ast.Token) bool {
	return isCapitalCase(token.Value)
}

func (p *Parser) expectConstraintIdentifier() ast.Token {
	token := p.expect(ast.TokenIdentifier)
	if !isPossibleConstraintIdentifier(token) {
		p.log.Fatalf("%s", parseErrorMessage(token, "Constraint must start with capital letter"))
	}
	return token
}

func (p *Parser) parseConstraintArgument() ast.ConstraintArgumentNode {
	token := p.current()
	switch {
	case token.Type == ast.TokenLBrace:
		shape := p.parseShapeLiteral(ShapeLiteralOpts{ParseAsTypes: true})
		return ast.ConstraintArgumentNode{Shape: &shape}
	case isPossibleTypeIdentifier(token, TypeIdentOpts{AllowLowercaseTypes: true}) ||
		token.Type == ast.TokenLBracket ||
		token.Type == ast.TokenMap ||
		token.Type == ast.TokenStar:
		typ := p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
		return ast.ConstraintArgumentNode{Type: &typ}
	default:
		value := p.parseValue()
		return ast.ConstraintArgumentNode{Value: &value}
	}
}

func (p *Parser) parseConstraint() ast.ConstraintNode {
	constraint := p.expectConstraintIdentifier()
	p.expect(ast.TokenLParen)

	var args []ast.ConstraintArgumentNode
	for p.current().Type != ast.TokenRParen {
		args = append(args, p.parseConstraintArgument())
		if p.current().Type == ast.TokenComma {
			p.advance()
		}
	}
	rparen := p.expect(ast.TokenRParen)

	return ast.ConstraintNode{
		Name: constraint.Value,
		Args: args,
		Span: ast.SpanBetweenTokens(constraint, rparen),
	}
}

func (p *Parser) rejectPipeInAssertion() {
	if p.current().Type == ast.TokenBitwiseOr {
		p.FailWithReport(p.current(), "refinement-pipe-in-assertion", "`|` is not valid inside `is`",
			"`|` is not valid inside `is`; use `or` to join assertion alternatives.",
			"write `is A() or B()`, not `is A() | B()`")
	}
}

// Helper to test builtin type token
func isBuiltinTypeToken(token ast.Token) bool {
	switch token.Type {
	case ast.TokenString, ast.TokenInt, ast.TokenFloat, ast.TokenBool:
		return true
	default:
		return false
	}
}

// Helper to check for shape literal refinement target
func (p *Parser) handleShapeRefinementTarget() (handled bool, target ast.RefinementTarget, assertion ast.AssertionNode) {
	if p.current().Type == ast.TokenLBrace {
		shape := p.parseShapeLiteral(ShapeLiteralOpts{})
		assertion := ast.AssertionNode{
			Constraints: []ast.ConstraintNode{{
				Name: "Match",
				Args: []ast.ConstraintArgumentNode{{Shape: &shape}},
			}},
		}
		assertion = p.parseAssertionOrJoin(assertion)
		return true, ast.AssertionTarget{Chains: assertion.MeetChains()}, assertion
	}
	return false, nil, ast.AssertionNode{}
}

// Helper to check for Shape(...) sugar
func (p *Parser) handleShapeSugar() (handled bool, target ast.RefinementTarget, assertion ast.AssertionNode) {
	if p.current().Type == ast.TokenIdentifier && p.current().Value == "Shape" && p.peek().Type == ast.TokenLParen {
		p.advance()
		p.expect(ast.TokenLParen)
		var shape ast.ShapeNode
		if p.current().Type == ast.TokenLBrace {
			shape = p.parseShapeLiteral(ShapeLiteralOpts{})
		} else {
			p.FailWithReport(p.current(), "parse-expected", "expected shape literal in Shape(...)",
				"Shape(...) requires a shape literal `{ ... }` argument.",
				"write `is { field: Type }` or `is Shape({ field: Type })`")
		}
		p.expect(ast.TokenRParen)
		assertion := ast.AssertionNode{
			Constraints: []ast.ConstraintNode{{
				Name: "Match",
				Args: []ast.ConstraintArgumentNode{{Shape: &shape}},
			}},
		}
		assertion = p.parseAssertionOrJoin(assertion)
		return true, ast.AssertionTarget{Chains: assertion.MeetChains()}, assertion
	}
	return false, nil, ast.AssertionNode{}
}

// Helper to determine qualified type
func (p *Parser) resolveQualifiedType(baseType **ast.TypeIdent) {
	if p.current().Type == ast.TokenDot {
		nextToken := p.peek()
		isQualifiedType := isPossibleTypeIdentifier(nextToken, TypeIdentOpts{AllowLowercaseTypes: false}) &&
			p.peek(2).Type != ast.TokenLParen

		if isQualifiedType {
			p.advance() // Consume dot
			pkgType := p.parseType(TypeIdentOpts{AllowLowercaseTypes: false})
			qualifiedName := ast.TypeIdent(string(**baseType) + "." + string(pkgType.Ident))
			*baseType = &qualifiedName
		}
	}
}

// Helper to check assertion constraint after dot
func (p *Parser) handleDotConstraintChain(baseType *ast.TypeIdent) (handled bool, target ast.RefinementTarget, assertion ast.AssertionNode) {
	if p.current().Type == ast.TokenDot {
		assertion := ast.AssertionNode{BaseType: baseType}
		assertion.Constraints = p.parseDotConstraintChain()
		assertion = p.parseAssertionOrJoin(assertion)
		return true, ast.AssertionTarget{Chains: assertion.MeetChains()}, assertion
	}
	return false, nil, ast.AssertionNode{}
}

func (p *Parser) parseRefinementTarget() (ast.RefinementTarget, ast.AssertionNode) {
	p.rejectPipeInAssertion()

	// Early clause for shape literal
	if handled, target, assertion := p.handleShapeRefinementTarget(); handled {
		return target, assertion
	}
	// Early clause for Shape(...) sugar
	if handled, target, assertion := p.handleShapeSugar(); handled {
		return target, assertion
	}

	token := p.current()
	isIdentOrConstraint := token.Type == ast.TokenIdentifier || isPossibleConstraintIdentifier(token)
	isBuiltinKeyword := isBuiltinTypeToken(token)

	if !isIdentOrConstraint && !isBuiltinKeyword {
		p.FailWithReport(token, "refinement-expected-target", "expected type name or assertion after `is`",
			"After `is`, Forst expects a type name, constraint call, or shape literal.",
			"examples: `is String`, `is Min(0)`, `is { name: String }`")
	}

	// Bare constraint call: Name(...)
	if (isIdentOrConstraint || isBuiltinKeyword) && p.peek().Type == ast.TokenLParen {
		if isBuiltinKeyword {
			p.FailWithReport(token, "refinement-builtin-constraint", "builtin type cannot be used as a constraint call",
				"Built-in types like String or Int cannot be called as constraints.",
				"use a constraint name (CapitalCase), e.g. `is Min(0)` or `is String.Min(1)`")
		}
		assertion := p.parseAssertionMeetChain(false)
		assertion = p.parseAssertionOrJoin(assertion)
		return ast.AssertionTarget{Chains: assertion.MeetChains()}, assertion
	}

	// Builtin keyword type as base (String.Min(...)) or bare TypeTarget (Int)
	if isBuiltinKeyword {
		baseType := builtinTypeIdentFromToken(token)
		p.advance()
		if handled, target, assertion := p.handleDotConstraintChain(baseType); handled {
			return target, assertion
		}
		p.rejectPipeInAssertion()
		tt := ast.TypeTarget{Name: *baseType}
		return tt, ast.AssertionNode{BaseType: baseType}
	}

	// Identifier: TypeTarget, qualified type, or Base.Constraint(...)
	typ := p.parseType(TypeIdentOpts{AllowLowercaseTypes: false})
	baseType := &typ.Ident
	p.resolveQualifiedType(&baseType)

	if handled, target, assertion := p.handleDotConstraintChain(baseType); handled {
		return target, assertion
	}

	p.rejectPipeInAssertion()
	tt := ast.TypeTarget{Name: *baseType}
	if p.current().Type == ast.TokenOr {
		p.FailWithReport(p.current(), "refinement-or-mixed-target", "`or` cannot join type names",
			"`or` cannot join type names; declare a union type with `|` (e.g. `type Live = A | B`).",
			"use `is A() or B()` for constraints, or `type T = A | B` for type unions")
	}
	return tt, ast.AssertionNode{BaseType: baseType}
}

func (p *Parser) parseAssertionChain(requireBaseType bool) ast.AssertionNode {
	return p.parseAssertionMeetChain(requireBaseType)
}

func (p *Parser) parseAssertionMeetChain(requireBaseType bool) ast.AssertionNode {
	var constraints []ast.ConstraintNode
	var baseType *ast.TypeIdent

	token := p.current()
	isIdentOrConstraint := token.Type == ast.TokenIdentifier || isPossibleConstraintIdentifier(token)
	isBuiltinKeyword := isBuiltinTypeToken(token)

	if isIdentOrConstraint || isBuiltinKeyword {
		if p.peek().Type == ast.TokenLParen {
			if requireBaseType {
				p.FailWithReport(token, "parse-expected", "expected base type for assertion",
					"A parenthesized assertion chain needs a base type before constraints.",
					"write `(Type.Constraint(...))` or use a bare constraint call")
			}
			constraint := p.parseConstraint()
			constraints = append(constraints, constraint)
		} else if isBuiltinKeyword {
			baseType = builtinTypeIdentFromToken(token)
			p.advance()
		} else {
			typ := p.parseType(TypeIdentOpts{AllowLowercaseTypes: false})
			baseType = &typ.Ident
			p.resolveQualifiedType(&baseType)
		}
	}

	constraints = append(constraints, p.parseDotConstraintChain()...)

	return ast.AssertionNode{
		BaseType:    baseType,
		Constraints: constraints,
	}
}

func (p *Parser) parseDotConstraintChain() []ast.ConstraintNode {
	var constraints []ast.ConstraintNode
	for p.current().Type == ast.TokenDot {
		p.advance()
		constraint := p.parseConstraint()
		constraints = append(constraints, constraint)
	}
	return constraints
}

func looksLikeConstraint(altTok ast.Token, peekTok ast.Token) bool {
	return (altTok.Type == ast.TokenIdentifier || isPossibleConstraintIdentifier(altTok)) &&
		peekTok.Type == ast.TokenLParen
}

func looksLikeBaseConstraint(altTok ast.Token, peekTok ast.Token) bool {
	switch altTok.Type {
	case ast.TokenIdentifier, ast.TokenString, ast.TokenInt, ast.TokenFloat, ast.TokenBool:
		return peekTok.Type == ast.TokenDot
	default:
		return false
	}
}

func (p *Parser) parseAssertionOrJoin(first ast.AssertionNode) ast.AssertionNode {
	for p.current().Type == ast.TokenOr {
		orTok := p.current()
		p.advance()
		p.rejectPipeInAssertion()

		if p.current().Type == ast.TokenIdentifier && p.peek().Type == ast.TokenIs {
			p.FailWithReport(p.current(), "refinement-repeated-is-subject", "do not repeat the subject after `or`",
				"do not repeat the subject after `or`; write `is A() or B()`.",
				"the subject is already bound — only list constraint alternatives after `or`")
		}

		altTok := p.current()
		peekTok := p.peek()

		if altTok.Type == ast.TokenBitwiseOr {
			p.rejectPipeInAssertion()
		}

		isConstraint := looksLikeConstraint(altTok, peekTok)
		isBaseConstraint := looksLikeBaseConstraint(altTok, peekTok)

		if isConstraint && !isPossibleConstraintIdentifier(altTok) {
			p.FailWithReport(altTok, "refinement-or-non-constraint", "`or` alternative must be a constraint chain",
				"`or` alternative must be a constraint chain with parentheses (constraint names are CapitalCase).",
				"write `is Foo() or Bar()` with CapitalCase constraint names")
		}

		if !isConstraint && !isBaseConstraint {
			if altTok.Type == ast.TokenIdentifier && peekTok.Type != ast.TokenLParen {
				p.FailWithReport(altTok, "refinement-or-non-constraint", "`or` alternative must be a constraint chain",
					fmt.Sprintf("`or` alternative must be a constraint chain with parentheses (got %q); use `else` for typed failure.", altTok.Value),
					"constraint alternatives need parentheses, e.g. `is A() or B()`")
			}
			p.FailWithReport(orTok, "refinement-or-non-constraint", "`or` joins assertion chains only",
				"`or` joins assertion constraint chains only, not boolean expressions.",
				"use `||` in expressions; use `or` only inside `is` refinements")
		}

		alt := p.parseAssertionMeetChain(false)
		if len(alt.Constraints) == 0 {
			p.FailWithReport(altTok, "refinement-or-mixed-target", "`or` cannot join a type name with an assertion",
				"`or` cannot join a type name with an assertion; use constraint chains only.",
				"write `is Type.Min(0) or Type.Max(10)`, not `is Type or Min(0)`")
		}
		first.OrChains = append(first.OrChains, alt)
	}
	p.rejectPipeInAssertion()
	return first
}

func builtinTypeIdentFromToken(token ast.Token) *ast.TypeIdent {
	var ident ast.TypeIdent
	switch token.Type {
	case ast.TokenString:
		ident = ast.TypeString
	case ast.TokenInt:
		ident = ast.TypeInt
	case ast.TokenFloat:
		ident = ast.TypeFloat
	case ast.TokenBool:
		ident = ast.TypeBool
	default:
		return nil
	}
	return &ident
}
