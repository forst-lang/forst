package lsp

import (
	"strconv"
	"unicode/utf8"

	"forst/internal/ast"
	"forst/internal/nodeinterop"
)

func lspLocationPtrFromAbsPath(absPath string, loc nodeinterop.IndexSourceLocation) *LSPLocation {
	if absPath == "" {
		return nil
	}
	uri := fileURIForLocalPath(absPath)
	if !loc.IsSet() {
		return &LSPLocation{
			URI: uri,
			Range: LSPRange{
				Start: LSPPosition{Line: 0, Character: 0},
				End:   LSPPosition{Line: 0, Character: 0},
			},
		}
	}
	line0 := max(loc.Line-1, 0)
	col0 := max(loc.Column, 0)
	endLine0 := line0
	endCol0 := col0 + 1
	if loc.EndLine > 0 {
		endLine0 = max(loc.EndLine-1, line0)
	}
	if loc.EndColumn > col0 {
		endCol0 = loc.EndColumn
	} else if loc.EndColumn == col0 {
		endCol0 = col0 + 1
	}
	return &LSPLocation{
		URI: uri,
		Range: LSPRange{
			Start: LSPPosition{Line: line0, Character: col0},
			End:   LSPPosition{Line: endLine0, Character: endCol0},
		},
	}
}

func lspLocationPtrFromExportName(absPath string, loc nodeinterop.IndexSourceLocation, exportName string) *LSPLocation {
	if loc.IsSet() {
		return lspLocationPtrFromAbsPath(absPath, loc)
	}
	width := max(utf8.RuneCountInString(exportName), 1)
	return &LSPLocation{
		URI: fileURIForLocalPath(absPath),
		Range: LSPRange{
			Start: LSPPosition{Line: 0, Character: 0},
			End:   LSPPosition{Line: 0, Character: width},
		},
	}
}

func (s *LSPServer) definingLocationForNodeImportPath(ctx *forstDocumentContext, tok *ast.Token) *LSPLocation {
	if ctx == nil || ctx.TC == nil || tok == nil || tok.Type != ast.TokenStringLiteral {
		return nil
	}
	i := tokenSliceIndex(ctx.Tokens, tok)
	if i < 0 || !importPathStringInImportClause(ctx.Tokens, i) {
		return nil
	}
	rawPath := tok.Value
	if unquoted, err := strconv.Unquote(rawPath); err == nil {
		rawPath = unquoted
	}
	abs, ok := ctx.TC.NodeImportPathAbsPath(rawPath)
	if !ok {
		return nil
	}
	return lspLocationPtrFromAbsPath(abs, nodeinterop.IndexSourceLocation{})
}

func (s *LSPServer) definingLocationForQualifiedNodeImport(ctx *forstDocumentContext, tokIdx int) *LSPLocation {
	moduleLocal, exportName, ok := qualifiedImportSymbolAt(ctx.Tokens, tokIdx)
	if !ok || ctx.TC == nil || !ctx.TC.IsNodeImportLocal(moduleLocal) {
		return nil
	}
	abs, loc, ok := ctx.TC.NodeExportDefinitionLocation(moduleLocal, exportName)
	if !ok {
		return nil
	}
	return lspLocationPtrFromExportName(abs, loc, exportName)
}

func (s *LSPServer) definingLocationForNodeImportLocal(ctx *forstDocumentContext, tokIdx int, tok *ast.Token) *LSPLocation {
	if ctx == nil || ctx.TC == nil || tok == nil || tok.Type != ast.TokenIdentifier {
		return nil
	}
	tc := ctx.TC
	moduleLocal := tok.Value
	if !tc.IsNodeImportLocal(moduleLocal) {
		return nil
	}
	if tokIdx+2 < len(ctx.Tokens) &&
		ctx.Tokens[tokIdx+1].Type == ast.TokenDot &&
		ctx.Tokens[tokIdx+2].Type == ast.TokenIdentifier {
		abs, ok := tc.NodeImportAbsPath(moduleLocal)
		if !ok {
			return nil
		}
		return lspLocationPtrFromAbsPath(abs, nodeinterop.IndexSourceLocation{})
	}
	if nodeImportAliasBindingAt(ctx.Tokens, tokIdx) {
		abs, ok := tc.NodeImportAbsPath(moduleLocal)
		if !ok {
			return nil
		}
		return lspLocationPtrFromAbsPath(abs, nodeinterop.IndexSourceLocation{})
	}
	return nil
}
