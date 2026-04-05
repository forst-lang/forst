package lsp

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"forst/internal/ast"
)

// handleCodeLens implements textDocument/codeLens: reference counts on top-level func/type/type-guard
// declarations in the current file, with a command the VS Code extension resolves to show references.
func (s *LSPServer) handleCodeLens(request LSPRequest) LSPServerResponse {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(request.Params, &params); err != nil {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &LSPError{
				Code:    -32602,
				Message: "Invalid params",
			},
		}
	}
	if params.TextDocument.URI == "" {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &LSPError{
				Code:    -32602,
				Message: "Invalid params: missing textDocument.uri",
			},
		}
	}

	lenses := s.codeLensesForURI(params.TextDocument.URI)
	if lenses == nil {
		lenses = []interface{}{}
	}
	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  lenses,
	}
}

type declLens struct {
	tok *ast.Token
}

func (s *LSPServer) codeLensesForURI(uri string) []interface{} {
	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		return nil
	}

	tc := ctx.TC
	var decls []declLens

	for name := range tc.Functions {
		if tok := findFuncNameToken(ctx.Tokens, string(name)); tok != nil {
			decls = append(decls, declLens{tok: tok})
		}
	}
	for name, def := range tc.Defs {
		if strings.HasPrefix(string(name), "T_") {
			continue
		}
		switch def.(type) {
		case ast.TypeDefNode:
			if tok := findTypeNameToken(ctx.Tokens, string(name)); tok != nil {
				decls = append(decls, declLens{tok: tok})
			}
		case ast.TypeGuardNode, *ast.TypeGuardNode:
			if tok := findTypeGuardNameToken(ctx.Tokens, string(name)); tok != nil {
				decls = append(decls, declLens{tok: tok})
			}
		}
	}

	sort.Slice(decls, func(i, j int) bool {
		ti, tj := decls[i].tok, decls[j].tok
		if ti.Line != tj.Line {
			return ti.Line < tj.Line
		}
		return ti.Column < tj.Column
	})

	out := make([]interface{}, 0, len(decls))
	for _, d := range decls {
		pos := lspPositionFromTokenStart(d.tok)
		locs := s.findReferencesFromContext(ctx, uri, pos, true)
		n := len(locs)
		title := fmt.Sprintf("%d references", n)
		if n == 1 {
			title = "1 reference"
		}
		rng := lspRangeFromToken(d.tok)
		out = append(out, map[string]interface{}{
			"range": map[string]interface{}{
				"start": map[string]interface{}{
					"line":      rng.Start.Line,
					"character": rng.Start.Character,
				},
				"end": map[string]interface{}{
					"line":      rng.End.Line,
					"character": rng.End.Character,
				},
			},
			"command": map[string]interface{}{
				"title":   title,
				"command": "forst.showReferences",
				"arguments": []interface{}{
					uri,
					pos.Line,
					pos.Character,
				},
			},
		})
	}
	return out
}

func lspPositionFromTokenStart(t *ast.Token) LSPPosition {
	if t == nil {
		return LSPPosition{}
	}
	line0 := t.Line - 1
	col0 := t.Column - 1
	if line0 < 0 {
		line0 = 0
	}
	if col0 < 0 {
		col0 = 0
	}
	return LSPPosition{Line: line0, Character: col0}
}

func lspRangeFromToken(t *ast.Token) LSPRange {
	loc := lspLocationFromToken("", t)
	return loc.Range
}
