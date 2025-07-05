package parser

import (
	"forst/internal/ast"
	"strconv"
)

func (p *Parser) parseImport() ast.ImportNode {
	var alias string
	if p.current().Type == ast.TokenIdentifier {
		alias = p.current().Value
		p.advance() // Skip alias
	}
	pathToken := p.expect(ast.TokenStringLiteral)
	rawPath := pathToken.Value
	path := rawPath
	if unquoted, err := strconv.Unquote(rawPath); err == nil {
		path = unquoted
	}

	node := ast.ImportNode{
		Path: path,
	}
	if alias != "" {
		node.Alias = &ast.Ident{ID: ast.Identifier(alias)}
	}

	return node
}

func (p *Parser) parseImportGroup() ast.ImportGroupNode {
	p.advance() // Move past '('
	imports := []ast.ImportNode{}

	// Parse imports until we hit the closing paren
	for p.current().Type != ast.TokenRParen {
		imports = append(imports, p.parseImport())
	}

	p.expect(ast.TokenRParen)
	return ast.ImportGroupNode{Imports: imports}
}

func (p *Parser) parseImports() []ast.Node {
	nodes := []ast.Node{}

	p.advance() // Move past `import`

	// Check if this is a grouped import with parentheses
	if p.current().Type == ast.TokenLParen {
		importGroup := p.parseImportGroup()
		p.logParsedNodeWithMessage(importGroup, "Parsed import group")
		nodes = append(nodes, importGroup)
	} else {
		importNode := p.parseImport()
		p.logParsedNodeWithMessage(importNode, "Parsed import")
		nodes = append(nodes, importNode)
	}

	return nodes
}
