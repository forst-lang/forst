package parser

import (
	"forst/pkg/ast"
)

// Parse a type definition
func (p *Parser) parseType() ast.TypeNode {
	token := p.current()

	switch token.Type {
	case ast.TokenString:
		p.advance()
		return ast.TypeNode{Name: ast.TypeString}
	case ast.TokenInt:
		p.advance()
		return ast.TypeNode{Name: ast.TypeInt}
	case ast.TokenFloat:
		p.advance()
		return ast.TypeNode{Name: ast.TypeFloat}
	case ast.TokenBool:
		p.advance()
		return ast.TypeNode{Name: ast.TypeBool}
	case ast.TokenVoid:
		p.advance()
		return ast.TypeNode{Name: ast.TypeVoid}
	default:
		typeName := p.expect(ast.TokenIdentifier).Value
		return ast.TypeNode{Name: typeName}
	}
}
