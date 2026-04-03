package lsp

import (
	"encoding/json"

	"forst/internal/ast"
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
	defTok := definingTokenForNavigableSymbol(ctx.TC, ctx.Tokens, tok)
	if defTok == nil {
		return nil
	}
	return collectIdentifierReferences(uri, ctx.Tokens, tok.Value, defTok, includeDecl)
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
