package parser

import (
	"forst/pkg/ast"
)

func (p *Parser) parsePackage() ast.PackageNode {
	p.expect(ast.TokenPackage) // Move past `package`

	packageName := p.expect(ast.TokenIdentifier).Value

	if packageName == "main" {
		p.context.IsMainPackage = true
	}

	return ast.PackageNode{Ident: ast.Ident{Id: ast.Identifier(packageName)}}
}
