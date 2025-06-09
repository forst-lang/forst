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
			p.logParsedNodeWithMessage(packageDef, "Parsed package")
			nodes = append(nodes, packageDef)
		case ast.TokenImport:
			nodes = append(nodes, p.parseImports()...)
		case ast.TokenType:
			typeDef := p.parseTypeDef()
			p.logParsedNodeWithMessage(typeDef, "Parsed type def")
			nodes = append(nodes, *typeDef)
		case ast.TokenFunc:
			scope := NewScope("", false, false, p.log)
			p.context.ScopeStack.PushScope(scope)
			function := p.parseFunctionDefinition()
			p.logParsedNodeWithMessage(function, "Parsed function")
			scope.FunctionName = string(function.Ident.ID)
			nodes = append(nodes, function)
		case ast.TokenIs:
			scope := NewScope("", false, true, p.log)
			p.context.ScopeStack.PushScope(scope)
			typeGuard := p.parseTypeGuard()
			p.logParsedNodeWithMessage(typeGuard, "Parsed type guard")
			scope.FunctionName = string(typeGuard.Ident)
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
