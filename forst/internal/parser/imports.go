package parser

import (
	"forst/internal/ast"
	"strconv"
	"strings"
)

func (p *Parser) parseImport(inGroup bool) ast.ImportNode {
	var alias *ast.Ident
	var sideEffectOnly bool
	var nodeOptIn bool
	var nodeOptInSource string
	var legacyPrefixNode bool

	switch p.current().Type {
	case ast.TokenStar:
		p.FailWithParseError(p.current(),
			`import * as … from is removed; use import "./path" node or import alias "./path" node`)
	case ast.TokenDot:
		// Go dot-import: import . "path" — symbols from path are in the file scope unqualified.
		p.advance()
		alias = &ast.Ident{ID: "."}
	case ast.TokenIdentifier:
		id := p.current().Value
		p.advance()
		if id == "node" && isLegacyPrefixNodeImport(p) {
			legacyPrefixNode = true
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

	if legacyPrefixNode {
		if p.current().Type == ast.TokenIdentifier && p.current().Value == "node" && !inGroup {
			p.FailWithParseError(p.current(),
				`cannot use both prefix and postfix node import marker; use import "./path" node or import alias "./path" node`)
		}
		nodeOptIn = true
		nodeOptInSource = "import_node_prefix"
	} else if p.current().Type == ast.TokenIdentifier && p.current().Value == "node" {
		if allowsPostfixNodeMarker(path, inGroup) {
			p.advance()
			nodeOptIn = true
			nodeOptInSource = "import_node"
		}
	}

	return ast.ImportNode{
		Path:            path,
		Alias:           alias,
		SideEffectOnly:  sideEffectOnly,
		NodeOptIn:       nodeOptIn,
		NodeOptInSource: nodeOptInSource,
	}
}

// isLegacyPrefixNodeImport reports whether `import node …` uses deprecated prefix syntax.
// `import node "fmt"` is a Go alias; relative paths, explicit aliases, and scoped npm paths stay legacy.
func isLegacyPrefixNodeImport(p *Parser) bool {
	switch p.current().Type {
	case ast.TokenIdentifier:
		return true
	case ast.TokenStringLiteral:
		return isLegacyNodeImportPath(unquoteImportPath(p.current().Value))
	default:
		return false
	}
}

func isLegacyNodeImportPath(path string) bool {
	return strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") || strings.HasPrefix(path, "@")
}

// allowsPostfixNodeMarker reports whether `node` after the import path is a postfix Node opt-in.
// In grouped imports, a bare Go-style path (e.g. "strconv") may be followed by a legacy `node "./path"` line.
func allowsPostfixNodeMarker(path string, inGroup bool) bool {
	if isLegacyNodeImportPath(path) {
		return true
	}
	if inGroup && isSingleSegmentImportPath(path) {
		return false
	}
	return true
}

func isSingleSegmentImportPath(path string) bool {
	return path != "" && !strings.Contains(path, "/")
}

func (p *Parser) parseImportGroup() ast.ImportGroupNode {
	p.advance() // Move past '('
	imports := []ast.ImportNode{}

	for p.current().Type != ast.TokenRParen {
		imp := p.parseImport(true)
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
		importNode := p.parseImport(false)
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
