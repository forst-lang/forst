package parser

import (
	"forst/internal/ast"
	"strconv"
)

func (p *Parser) parseImport() ast.ImportNode {
	var alias *ast.Ident
	var sideEffectOnly bool
	var nodeOptIn bool
	var nodeOptInSource string

	switch p.current().Type {
	case ast.TokenStar:
		p.FailWithParseError(p.current(),
			`import * as … from is removed; use import node "./path" or import node alias "./path"`)
	case ast.TokenDot:
		// Go dot-import: import . "path" — symbols from path are in the file scope unqualified.
		p.advance()
		alias = &ast.Ident{ID: "."}
	case ast.TokenIdentifier:
		id := p.current().Value
		p.advance()
		if id == "node" {
			nodeOptIn = true
			nodeOptInSource = "import_node"
			if p.current().Type == ast.TokenIdentifier {
				aliasID := p.current().Value
				p.advance()
				alias = &ast.Ident{ID: ast.Identifier(aliasID)}
			}
		} else {
			alias = &ast.Ident{ID: ast.Identifier(id)}
			if id == "_" {
				sideEffectOnly = true
			}
		}
	}

	pathToken := p.expect(ast.TokenStringLiteral)
	path := unquoteImportPath(pathToken.Value)

	return ast.ImportNode{
		Path:            path,
		Alias:           alias,
		SideEffectOnly:  sideEffectOnly,
		NodeOptIn:       nodeOptIn,
		NodeOptInSource: nodeOptInSource,
	}
}

func (p *Parser) parseImportGroup() ast.ImportGroupNode {
	p.advance() // Move past '('
	imports := []ast.ImportNode{}

	for p.current().Type != ast.TokenRParen {
		imp := p.parseImport()
		imports = append(imports, imp)
	}

	p.expect(ast.TokenRParen)
	return ast.ImportGroupNode{Imports: imports}
}

func (p *Parser) parseImports() []ast.Node {
	nodes := []ast.Node{}

	p.advance() // Move past `import`

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

func unquoteImportPath(raw string) string {
	if unquoted, err := strconv.Unquote(raw); err == nil {
		return unquoted
	}
	return raw
}
