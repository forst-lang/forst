package parser

import (
	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func (p *Parser) parseInlineTypeGuardBody(subjectParam ast.ParamNode) []ast.Node {
	// If the body is inline, parse it as an ensure statement
	ensure := p.parseEnsureStatement()
	if ensure.Variable.GetIdent() != subjectParam.GetIdent() {
		p.FailWithParseError(p.current(), "inline type guard must refine the subject parameter")
	}
	return []ast.Node{ensure}
}

// parseTypeGuardBody parses the body of a type guard
func (p *Parser) parseTypeGuardBody(subjectParam ast.ParamNode) []ast.Node {
	// If the body is a block, parse it
	if p.current().Type == ast.TokenLBrace {
		return p.parseBlock()
	}

	return p.parseInlineTypeGuardBody(subjectParam)
}

// parseTypeGuard parses a type guard declaration
func (p *Parser) parseTypeGuard() *ast.TypeGuardNode {
	p.expect(ast.TokenIs)
	p.expect(ast.TokenLParen)

	// Parse subject parameter (required)
	if p.current().Type == ast.TokenRParen {
		p.FailWithParseError(p.current(), "type guard requires a subject parameter")
	}
	p.log.WithFields(logrus.Fields{
		"function": "parseTypeGuard",
		"token":    p.current(),
	}).Trace("Parsing subject parameter")
	subjectParam := p.parseParameter()
	p.log.WithFields(logrus.Fields{
		"function":     "parseTypeGuard",
		"subjectParam": subjectParam,
		"token":        p.current(),
	}).Trace("Parsed subject parameter")
	p.expect(ast.TokenRParen)

	// Parse guard name and additional parameters if present
	var guardName ast.Identifier
	var additionalParams []ast.ParamNode
	if p.current().Type == ast.TokenIdentifier {
		guardName = ast.Identifier(p.current().Value)
		p.advance()
		p.log.WithFields(logrus.Fields{
			"function":  "parseTypeGuard",
			"guardName": guardName,
			"token":     p.current(),
		}).Trace("Parsed guard name")
		if p.current().Type == ast.TokenLParen {
			p.advance()
			// Parse additional parameters
			for p.current().Type != ast.TokenRParen {
				p.log.WithFields(logrus.Fields{
					"function": "parseTypeGuard",
					"token":    p.current(),
				}).Trace("Parsing additional parameter")
				param := p.parseParameter()
				p.log.WithFields(logrus.Fields{
					"function": "parseTypeGuard",
					"param":    param,
					"token":    p.current(),
				}).Trace("Parsed additional parameter")
				additionalParams = append(additionalParams, param)
				if p.current().Type == ast.TokenComma {
					p.advance()
				}
			}
			p.expect(ast.TokenRParen)
		}
	} else {
		p.FailWithParseError(p.current(), "expected guard name")
	}

	// Parse body - can be either a block or a single expression
	body := p.parseTypeGuardBody(subjectParam)

	return &ast.TypeGuardNode{
		Ident:   guardName,
		Subject: subjectParam,
		Params:  additionalParams,
		Body:    body,
	}
}
