package parser

import (
	"forst/pkg/ast"
)

func (p *Parser) parseImports() []ast.Node {
	nodes := []ast.Node{}

	p.advance() // Move past `import`

	// Check if this is a grouped import with parentheses
	if p.current().Type == ast.TokenLParen {
		p.advance() // Move past '('
		imports := []ast.ImportNode{}

		// Parse imports until we hit the closing paren
		for p.current().Type != ast.TokenRParen {
			var alias string
			if p.current().Type == ast.TokenIdentifier {
				alias = p.current().Value
				p.advance() // Skip alias
			}
			path := p.expect(ast.TokenStringLiteral).Value
			imports = append(imports, ast.ImportNode{Path: path, Alias: &alias})
		}

		p.expect(ast.TokenRParen)
		nodes = append(nodes, ast.ImportGroupNode{Imports: imports})
	} else {
		// Single import
		importName := p.expect(ast.TokenIdentifier).Value
		nodes = append(nodes, ast.ImportNode{Path: importName})
	}

	return nodes
}
