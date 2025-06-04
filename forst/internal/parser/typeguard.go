package parser

import (
	"forst/internal/ast"

	log "github.com/sirupsen/logrus"
)

func (p *Parser) parseInlineTypeGuardBody(subjectParam ast.ParamNode) []ast.Node {
	// If the body is inline, parse it as an ensure statement
	ensure := p.parseEnsureStatement()
	if ensure.Variable.GetIdent() != subjectParam.GetIdent() {
		panic(parseErrorMessage(p.current(), "inline type guard must refine the subject parameter"))
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
		panic(parseErrorMessage(p.current(), "type guard requires a subject parameter"))
	}
	log.Tracef("[parseTypeGuard] Parsing subject parameter, current token: %+v", p.current())
	subjectParam := p.parseParameter()
	log.Tracef("[parseTypeGuard] Parsed subject parameter: %+v, next token: %+v", subjectParam, p.current())
	p.expect(ast.TokenRParen)

	// Parse guard name and additional parameters if present
	var guardName ast.Identifier
	var additionalParams []ast.ParamNode
	if p.current().Type == ast.TokenIdentifier {
		guardName = ast.Identifier(p.current().Value)
		p.advance()
		log.Tracef("[parseTypeGuard] Parsed guard name: %s, current token: %+v", guardName, p.current())
		if p.current().Type == ast.TokenLParen {
			p.advance()
			// Parse additional parameters
			for p.current().Type != ast.TokenRParen {
				log.Tracef("[parseTypeGuard] Parsing additional parameter, current token: %+v", p.current())
				param := p.parseParameter()
				log.Tracef("[parseTypeGuard] Parsed additional parameter: %+v, next token: %+v", param, p.current())
				additionalParams = append(additionalParams, param)
				if p.current().Type == ast.TokenComma {
					p.advance()
				}
			}
			p.expect(ast.TokenRParen)
		}
	} else {
		panic(parseErrorMessage(p.current(), "expected guard name"))
	}

	// Parse body - can be either a block or a single expression
	body := p.parseTypeGuardBody(subjectParam)

	return &ast.TypeGuardNode{
		Ident:            guardName,
		SubjectParam:     subjectParam,
		AdditionalParams: additionalParams,
		Body:             body,
	}
}
