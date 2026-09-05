package parser

import "forst/internal/ast"

func (p *Parser) parseMakeNewCallArguments(builtin ast.Identifier) ([]ast.ExpressionNode, []ast.SourceSpan) {
	var args []ast.ExpressionNode
	var argSpans []ast.SourceSpan
	if p.current().Type == ast.TokenRParen {
		return args, argSpans
	}
	startTok := p.current()
	if p.current().Type != ast.TokenStar && !isPossibleTypeIdentifier(p.current(), TypeIdentOpts{AllowLowercaseTypes: true}) {
		p.FailWithReport(p.current(), "make-new-type-arg", string(builtin)+"() first argument must be a type",
			string(builtin)+"() first argument must be a type (e.g. Array(Int), map[String]Int, *Int).",
			"pass a type as the first argument to "+string(builtin))
	}
	typ := p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
	endTok := p.tokens[p.currentIndex-1]
	args = append(args, ast.TypeExpressionNode{Type: typ})
	argSpans = append(argSpans, ast.SpanBetweenTokens(startTok, endTok))
	for p.current().Type == ast.TokenComma {
		p.advance()
		startTok = p.current()
		arg := p.parseExpression()
		endTok = p.tokens[p.currentIndex-1]
		args = append(args, arg)
		argSpans = append(argSpans, ast.SpanBetweenTokens(startTok, endTok))
	}
	return args, argSpans
}
