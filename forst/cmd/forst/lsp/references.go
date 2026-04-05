package lsp

import (
	"encoding/json"

	"forst/internal/ast"
	"forst/internal/typechecker"
)

// handleReferences implements textDocument/references for same-file identifier occurrences
// of top-level functions, user types, and type guards (same resolution as go-to-definition).
func (s *LSPServer) handleReferences(request LSPRequest) LSPServerResponse {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Position LSPPosition `json:"position"`
		Context  struct {
			IncludeDeclaration bool `json:"includeDeclaration"`
		} `json:"context"`
	}
	if err := json.Unmarshal(request.Params, &params); err != nil {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &LSPError{
				Code:    -32700,
				Message: "Parse error",
			},
		}
	}

	locs := s.findReferencesForPosition(
		params.TextDocument.URI,
		params.Position,
		params.Context.IncludeDeclaration,
	)
	if locs == nil {
		locs = []LSPLocation{}
	}
	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  locs,
	}
}

func (s *LSPServer) findReferencesForPosition(uri string, position LSPPosition, includeDecl bool) []LSPLocation {
	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		return nil
	}
	tok := tokenAtLSPPosition(ctx.Tokens, position)
	if tok == nil || tok.Type != ast.TokenIdentifier {
		return nil
	}
	if ctx.PackageMerge != nil {
		if decl := s.definingTopLevelLocationForPackage(ctx.TC, uri, ctx.Tokens, tok, ctx.PackageMerge); decl != nil {
			return collectTopLevelReferencesAcrossPackage(ctx.PackageMerge, tok.Value, decl, includeDecl)
		}
	} else {
		if defTok := definingTokenForNavigableSymbol(ctx.TC, ctx.Tokens, tok); defTok != nil {
			return collectIdentifierReferences(uri, ctx.Tokens, tok.Value, defTok, includeDecl)
		}
	}
	tokIdx := tokenIndexAtLSPPosition(ctx.Tokens, position)
	if tokIdx < 0 {
		return nil
	}
	sym, ok := lookupSymbolAtToken(ctx.TC, ctx.Nodes, ctx.Tokens, tokIdx, ast.Identifier(tok.Value))
	if !ok {
		return nil
	}
	defTok := definingTokenForLocalBinding(ctx, tokIdx, tok)
	if defTok == nil {
		return nil
	}
	return collectIdentifierReferencesSameBinding(uri, ctx.Tokens, tok.Value, defTok, includeDecl, sym, ctx.TC, ctx.Nodes)
}

func collectTopLevelReferencesAcrossPackage(merge *packageMergeInfo, name string, decl *LSPLocation, includeDecl bool) []LSPLocation {
	if merge == nil || decl == nil {
		return nil
	}
	declURI := decl.URI
	declLine := decl.Range.Start.Line + 1
	declCol := decl.Range.Start.Character + 1

	var out []LSPLocation
	for _, u := range merge.MemberURIs {
		tokens := merge.TokensByURI[u]
		for i := range tokens {
			t := &tokens[i]
			if t.Type != ast.TokenIdentifier || t.Value != name {
				continue
			}
			if !includeDecl && u == declURI && int(t.Line) == declLine && int(t.Column) == declCol {
				continue
			}
			out = append(out, lspLocationFromToken(u, t))
		}
	}
	return out
}

func collectIdentifierReferencesSameBinding(uri string, tokens []ast.Token, name string, defTok *ast.Token, includeDecl bool, refSym typechecker.Symbol, tc *typechecker.TypeChecker, nodes []ast.Node) []LSPLocation {
	var out []LSPLocation
	for i := range tokens {
		t := &tokens[i]
		if t.Type != ast.TokenIdentifier || t.Value != name {
			continue
		}
		sym2, ok2 := lookupSymbolAtToken(tc, nodes, tokens, i, ast.Identifier(name))
		if !ok2 || !sameBinding(sym2, refSym) {
			continue
		}
		if !includeDecl && tokenSamePosition(t, defTok) {
			continue
		}
		out = append(out, lspLocationFromToken(uri, t))
	}
	return out
}

func collectIdentifierReferences(uri string, tokens []ast.Token, name string, defTok *ast.Token, includeDecl bool) []LSPLocation {
	var out []LSPLocation
	for i := range tokens {
		t := &tokens[i]
		if t.Type != ast.TokenIdentifier || t.Value != name {
			continue
		}
		if !includeDecl && tokenSamePosition(t, defTok) {
			continue
		}
		out = append(out, lspLocationFromToken(uri, t))
	}
	return out
}

func tokenSamePosition(a, b *ast.Token) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Line == b.Line && a.Column == b.Column
}
