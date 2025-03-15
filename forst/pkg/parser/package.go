package parser

import (
	"forst/pkg/ast"
)

func (p *Parser) parsePackage(context *Context) ast.PackageNode {
	p.expect(ast.TokenPackage) // Move past `package`

	packageName := p.expect(ast.TokenIdentifier).Value

	if packageName == "main" {
		context.IsMainPackage = true
	}

	return ast.PackageNode{Value: packageName}
}
