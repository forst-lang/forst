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

// Scope represents a scope in the parsed program
type Scope struct {
	// All variables defined in the scope
	Variables map[string]ast.TypeNode
	// The function currently being parsed
	functionName string
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
		case ast.TokenFunction:
			p.context.Scope = &Scope{
				Variables: make(map[string]ast.TypeNode),
			}
			function := p.parseFunctionDefinition()
			logParsedNodeWithMessage(function, "Parsed function")
			nodes = append(nodes, function)
		case ast.TokenIs:
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

// DefineVariable defines a variable in the scope
func (s *Scope) DefineVariable(name string, typeNode ast.TypeNode) {
	s.Variables[name] = typeNode
}

// IsMainFunction checks if the current function is the main function
func (c *Context) IsMainFunction() bool {
	if c.Package == nil || !c.Package.IsMainPackage() {
		return false
	}

	if c.Scope.functionName != "main" {
		return true
	}

	return true
}
