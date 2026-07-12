package lsp

import (
	"strconv"
	"unicode/utf8"

	"forst/internal/ast"
	"forst/internal/goload"
)

func lspLocationPtrFromGoSource(loc goload.SourceLocation) *LSPLocation {
	if loc.File == "" {
		return nil
	}
	uri := fileURIForLocalPath(loc.File)
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

func lspLocationPtrFromGoSymbol(loc goload.SourceLocation, symbol string) *LSPLocation {
	if loc.IsSet() {
		return lspLocationPtrFromGoSource(loc)
	}
	if loc.File == "" {
		return nil
	}
	width := max(utf8.RuneCountInString(symbol), 1)
	return &LSPLocation{
		URI: fileURIForLocalPath(loc.File),
		Range: LSPRange{
			Start: LSPPosition{Line: 0, Character: 0},
			End:   LSPPosition{Line: 0, Character: width},
		},
	}
}

func (s *LSPServer) definingLocationForGoImportPath(ctx *forstDocumentContext, tok *ast.Token) *LSPLocation {
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
	path := goload.ImportPathFromForst(rawPath)
	loc, ok := ctx.TC.GoImportPathDefinitionLocation(path)
	if !ok {
		return nil
	}
	return lspLocationPtrFromGoSource(loc)
}

func (s *LSPServer) definingLocationForQualifiedGoImport(ctx *forstDocumentContext, tokIdx int) *LSPLocation {
	pkgLocal, symbol, ok := qualifiedImportSymbolAt(ctx.Tokens, tokIdx)
	if !ok || ctx.TC == nil {
		return nil
	}
	if goImportLocalShadowedByForstVar(ctx.TC, pkgLocal) || !ctx.TC.IsImportedLocalName(pkgLocal) || ctx.TC.IsNodeImportLocal(pkgLocal) {
		return nil
	}
	loc, ok := ctx.TC.GoQualifiedExportDefinitionLocation(pkgLocal, symbol)
	if !ok {
		return nil
	}
	return lspLocationPtrFromGoSymbol(loc, symbol)
}

func (s *LSPServer) definingLocationForGoImportLocal(ctx *forstDocumentContext, tokIdx int, tok *ast.Token) *LSPLocation {
	if ctx == nil || ctx.TC == nil || tok == nil || tok.Type != ast.TokenIdentifier {
		return nil
	}
	tc := ctx.TC
	pkgLocal := tok.Value
	if goImportLocalShadowedByForstVar(tc, pkgLocal) || !tc.IsImportedLocalName(pkgLocal) || tc.IsNodeImportLocal(pkgLocal) {
		return nil
	}
	if tokIdx+2 < len(ctx.Tokens) &&
		ctx.Tokens[tokIdx+1].Type == ast.TokenDot &&
		ctx.Tokens[tokIdx+2].Type == ast.TokenIdentifier {
		loc, ok := tc.GoImportLocalDefinitionLocation(pkgLocal)
		if !ok {
			return nil
		}
		return lspLocationPtrFromGoSource(loc)
	}
	return nil
}

func (s *LSPServer) definingLocationForGoReceiverMethod(ctx *forstDocumentContext, tokIdx int) *LSPLocation {
	recvLocal, methodName, ok := qualifiedImportSymbolAt(ctx.Tokens, tokIdx)
	if !ok || ctx.TC == nil {
		return nil
	}
	// Skip when recv is a Go import local without a Forst shadow (e.g. fmt.Println). Forst locals
	// bound from Go calls (e.g. cmd from exec.Command) still resolve receiver methods here.
	if ctx.TC.IsImportedLocalName(recvLocal) && !goImportLocalShadowedByForstVar(ctx.TC, recvLocal) {
		return nil
	}
	if ctx.TC.IsNodeImportLocal(recvLocal) {
		return nil
	}
	loc, ok := ctx.TC.GoReceiverMethodDefinitionLocation(recvLocal, methodName)
	if !ok {
		return nil
	}
	return lspLocationPtrFromGoSymbol(loc, methodName)
}

func (s *LSPServer) definingLocationForGoUnqualifiedExport(ctx *forstDocumentContext, tok *ast.Token) *LSPLocation {
	if ctx == nil || ctx.TC == nil || tok == nil || tok.Type != ast.TokenIdentifier {
		return nil
	}
	symbol := tok.Value
	if loc, ok := ctx.TC.GoSamePackageFuncDefinitionLocation(symbol); ok {
		return lspLocationPtrFromGoSymbol(loc, symbol)
	}
	if loc, ok := ctx.TC.GoDotImportFuncDefinitionLocation(symbol); ok {
		return lspLocationPtrFromGoSymbol(loc, symbol)
	}
	return nil
}
