package parser

import (
	"fmt"

	"forst/pkg/ast"
)

// Parse the tokens into an AST
func (p *Parser) Parse() []ast.Node {
	nodes := []ast.Node{}

	for p.current().Type != ast.TokenEOF {
		if p.current().Type == ast.TokenPackage {
			p.advance() // Move past `package`
			packageName := p.expect(ast.TokenIdentifier).Value
			nodes = append(nodes, ast.PackageNode{Value: packageName})
		} else if p.current().Type == ast.TokenImport {
			p.advance() // Move past `import`

			// Check if this is a grouped import with parentheses
			if p.current().Type == ast.TokenLParen {
				p.advance() // Move past '('
				imports := []ast.ImportNode{}

				// Parse imports until we hit the closing paren
				for p.current().Type != ast.TokenRParen {
					importName := p.expect(ast.TokenIdentifier).Value
					imports = append(imports, ast.ImportNode{Path: importName})
				}

				p.expect(ast.TokenRParen)
				nodes = append(nodes, ast.ImportGroupNode{Imports: imports})
			} else {
				// Single import
				importName := p.expect(ast.TokenIdentifier).Value
				nodes = append(nodes, ast.ImportNode{Path: importName})
			}
		} else if p.current().Type == ast.TokenFunction {
			nodes = append(nodes, p.parseFunctionDefinition())
		} else {
			panic(fmt.Sprintf("Unexpected token in file: %s", p.current().Value))
		}
	}

	return nodes
}
