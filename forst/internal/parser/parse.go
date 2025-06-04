// Package parser converts tokens into Forst AST nodes.
package parser

import (
	"fmt"

	"forst/internal/ast"
)

// ParseError represents a parse error
type ParseError struct {
	Token   ast.Token
	Context *Context
	Msg     string
}

// Location returns the location of the parse error
func (e *ParseError) Location() string {
	if e.Context != nil {
		return fmt.Sprintf("%s:%d:%d", e.Context.FilePath, e.Token.Line, e.Token.Column)
	}
	return fmt.Sprintf("line %d, column %d", e.Token.Line, e.Token.Column)
}

// Error returns the message of the parse error
func (e *ParseError) Error() string {
	return fmt.Sprintf("Parse error at %s: %s", e.Location(), e.Msg)
}

// ParseFile parses the tokens in a Forst file into a list of Forst AST nodes
func (p *Parser) ParseFile() ([]ast.Node, error) {
	nodes := []ast.Node{}

	for p.current().Type != ast.TokenEOF {
		token := p.current()

		switch token.Type {
		case ast.TokenComment:
			// Comments are ignored
			p.advance()
			continue
		case ast.TokenPackage:
			packageDef := p.parsePackage()
			logParsedNodeWithMessage(packageDef, "Parsed package")
			nodes = append(nodes, packageDef)
		case ast.TokenImport:
			nodes = append(nodes, p.parseImports()...)
		case ast.TokenType:
			typeDef := p.parseTypeDef()
			logParsedNodeWithMessage(typeDef, "Parsed type def")
			nodes = append(nodes, *typeDef)
		case ast.TokenFunc:
			p.context.ScopeStack.PushScope(&Scope{
				Variables:   make(map[string]ast.TypeNode),
				IsTypeGuard: false,
			})
			function := p.parseFunctionDefinition()
			logParsedNodeWithMessage(function, "Parsed function")
			nodes = append(nodes, function)
		case ast.TokenIs:
			p.context.ScopeStack.PushScope(&Scope{
				Variables:   make(map[string]ast.TypeNode),
				IsTypeGuard: true,
			})
			typeGuard := p.parseTypeGuard()
			logParsedNodeWithMessage(typeGuard, "Parsed type guard")
			nodes = append(nodes, typeGuard)
		default:
			return nil, &ParseError{
				Token:   token,
				Context: p.context,
				Msg:     fmt.Sprintf("unexpected token: %s", token.Value),
			}
		}
	}

	return nodes, nil
}
