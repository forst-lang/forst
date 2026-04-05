package lsp

import (
	"encoding/json"
	"sort"
	"strings"

	"forst/internal/ast"
)

// handlePrepareRename implements textDocument/prepareRename for identifiers that support the same
// binding resolution as find references (locals, parameters, and merged same-package top-level symbols).
func (s *LSPServer) handlePrepareRename(request LSPRequest) LSPServerResponse {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Position LSPPosition `json:"position"`
	}
	if err := json.Unmarshal(request.Params, &params); err != nil {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error:   &LSPError{Code: -32602, Message: "Invalid params"},
		}
	}
	if params.TextDocument.URI == "" {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error:   &LSPError{Code: -32602, Message: "Invalid params: missing textDocument.uri"},
		}
	}
	ctx, ok := s.analyzeForstDocument(params.TextDocument.URI)
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		return LSPServerResponse{JSONRPC: "2.0", ID: request.ID, Result: nil}
	}
	tok := tokenAtLSPPosition(ctx.Tokens, params.Position)
	if tok == nil || tok.Type != ast.TokenIdentifier || strings.HasPrefix(tok.Value, "T_") {
		return LSPServerResponse{JSONRPC: "2.0", ID: request.ID, Result: nil}
	}
	locs := s.findReferencesFromContext(ctx, params.TextDocument.URI, params.Position, true)
	if len(locs) == 0 {
		return LSPServerResponse{JSONRPC: "2.0", ID: request.ID, Result: nil}
	}
	rng := lspLocationFromToken(params.TextDocument.URI, tok).Range
	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: LSPPrepareRenameResult{
			Range:       rng,
			Placeholder: tok.Value,
		},
	}
}

// handleRename implements textDocument/rename using the same occurrence set as references.
func (s *LSPServer) handleRename(request LSPRequest) LSPServerResponse {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Position LSPPosition `json:"position"`
		NewName  string      `json:"newName"`
	}
	if err := json.Unmarshal(request.Params, &params); err != nil {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error:   &LSPError{Code: -32602, Message: "Invalid params"},
		}
	}
	if params.TextDocument.URI == "" {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error:   &LSPError{Code: -32602, Message: "Invalid params: missing textDocument.uri"},
		}
	}
	name := strings.TrimSpace(params.NewName)
	if !isValidForstIdentifierRename(name) {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error:   &LSPError{Code: -32602, Message: "invalid newName"},
		}
	}
	ctx, ok := s.analyzeForstDocument(params.TextDocument.URI)
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		return LSPServerResponse{JSONRPC: "2.0", ID: request.ID, Result: nil}
	}
	tok := tokenAtLSPPosition(ctx.Tokens, params.Position)
	if tok == nil || tok.Type != ast.TokenIdentifier || strings.HasPrefix(tok.Value, "T_") {
		return LSPServerResponse{JSONRPC: "2.0", ID: request.ID, Result: nil}
	}
	oldName := tok.Value
	if oldName == name {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result:  LSPWorkspaceEdit{},
		}
	}
	locs := s.findReferencesFromContext(ctx, params.TextDocument.URI, params.Position, true)
	if len(locs) == 0 {
		return LSPServerResponse{JSONRPC: "2.0", ID: request.ID, Result: nil}
	}
	byURI := make(map[string][]LSPTextEdit)
	for _, loc := range locs {
		byURI[loc.URI] = append(byURI[loc.URI], LSPTextEdit{
			Range:   loc.Range,
			NewText: name,
		})
	}
	for u := range byURI {
		sort.Slice(byURI[u], func(i, j int) bool {
			a, b := byURI[u][i].Range.Start, byURI[u][j].Range.Start
			if a.Line != b.Line {
				return a.Line > b.Line
			}
			return a.Character > b.Character
		})
	}
	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: LSPWorkspaceEdit{
			Changes: byURI,
		},
	}
}

func isValidForstIdentifierRename(s string) bool {
	if s == "" {
		return false
	}
	r := []rune(s)
	first := r[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		return false
	}
	for _, c := range r[1:] {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}
