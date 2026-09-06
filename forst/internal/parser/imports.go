package parser

import (
	"forst/internal/ast"
	"strconv"
	"strings"
)

func (p *Parser) parseImport(inGroup bool) ast.ImportNode {
	var alias *ast.Ident
	var sideEffectOnly bool
	var bridgeOptIn bool
	var bridgeOptInSource string

	switch p.current().Type {
	case ast.TokenStar:
		p.FailWithReport(p.current(), "import-star-removed", "`import * as … from` is removed",
			"Wildcard imports are not supported.",
			`use import "./path" js or import alias "./path" js`)
	case ast.TokenDot:
		// Go dot-import: import . "path" — symbols from path are in the file scope unqualified.
		p.advance()
		alias = &ast.Ident{ID: "."}
	case ast.TokenIdentifier:
		id := p.current().Value
		tok := p.current()
		p.advance()
		alias = &ast.Ident{ID: ast.Identifier(id), Span: ast.SpanFromToken(tok)}
		if id == "_" {
			sideEffectOnly = true
		}
	}

	pathToken := p.expect(ast.TokenStringLiteral)
	path := unquoteImportPath(pathToken.Value)
	importSpan := ast.SpanFromToken(pathToken)
	if alias != nil && alias.Span.IsSet() {
		importSpan = ast.SpanBetweenTokens(
			ast.Token{Line: alias.Span.StartLine, Column: alias.Span.StartCol},
			pathToken,
		)
	}

	if p.current().Type == ast.TokenIdentifier {
		switch p.current().Value {
		case "js":
			if allowsPostfixJSMarker(path, inGroup) {
				p.advance()
				bridgeOptIn = true
				bridgeOptInSource = "import_js"
			}
		case "node":
			if allowsPostfixJSMarker(path, inGroup) {
				p.FailWithReport(p.current(), "import-node-removed", `postfix "node" import marker was removed`,
					`The "node" postfix import marker is no longer supported.`,
					`use import "./path" js or import alias "./path" js`)
			}
		}
	}

	return ast.ImportNode{
		Span:              importSpan,
		Path:              path,
		Alias:             alias,
		SideEffectOnly:    sideEffectOnly,
		BridgeOptIn:       bridgeOptIn,
		BridgeOptInSource: bridgeOptInSource,
	}
}

func isScriptImportPath(path string) bool {
	return strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") || strings.HasPrefix(path, "@")
}

// allowsPostfixJSMarker reports whether `js` after the import path is a postfix bridge opt-in.
// In grouped imports, a bare Go-style path (e.g. "strconv") may be followed by a `js "./path"` line.
func allowsPostfixJSMarker(path string, inGroup bool) bool {
	if isScriptImportPath(path) {
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
