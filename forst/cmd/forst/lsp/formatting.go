package lsp

import (
	"encoding/json"
	"strings"

	"forst/internal/printer"
)

// documentFormattingEdits returns a single full-buffer replace when formatting changes text, or nil if unchanged / unknown URI.
func (s *LSPServer) documentFormattingEdits(uri string, tabSize int, insertSpaces bool) []LSPTextEdit {
	src, ok := s.openDocumentText(uri)
	if !ok {
		return nil
	}
	if tabSize <= 0 {
		tabSize = 4
	}
	formatted := printer.FormatDocument(src, uri, tabSize, insertSpaces, s.log)
	if formatted == src {
		return nil
	}
	end := printer.EndPositionExclusive(src)
	return []LSPTextEdit{
		{
			Range: LSPRange{
				Start: LSPPosition{Line: 0, Character: 0},
				End:   LSPPosition{Line: end.Line, Character: end.Character},
			},
			NewText: formatted,
		},
	}
}

// handleFormatting handles the textDocument/formatting method
func (s *LSPServer) handleFormatting(request LSPRequest) LSPServerResponse {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Options struct {
			TabSize      int  `json:"tabSize"`
			InsertSpaces bool `json:"insertSpaces"`
		} `json:"options"`
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

	tabSize := params.Options.TabSize
	if tabSize <= 0 {
		tabSize = 4
	}
	edits := s.documentFormattingEdits(params.TextDocument.URI, tabSize, params.Options.InsertSpaces)
	if edits == nil {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result:  nil,
		}
	}
	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  edits,
	}
}

// codeActionKindsAllowFormat returns true when context.only is empty or includes a `source` kind.
func codeActionKindsAllowFormat(only []string) bool {
	if len(only) == 0 {
		return true
	}
	for _, k := range only {
		if k == "source" || k == "source.formatDocument" || strings.HasPrefix(k, "source.") {
			return true
		}
	}
	return false
}

// handleCodeAction handles the textDocument/codeAction method
func (s *LSPServer) handleCodeAction(request LSPRequest) LSPServerResponse {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Context struct {
			Only []string `json:"only"`
		} `json:"context"`
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

	var out []interface{}
	if codeActionKindsAllowFormat(params.Context.Only) {
		if edits := s.documentFormattingEdits(params.TextDocument.URI, 4, true); edits != nil {
			out = append(out, LSPCodeAction{
				Title: "Format document",
				Kind:  "source.formatDocument",
				Edit: &LSPWorkspaceEdit{
					Changes: map[string][]LSPTextEdit{
						params.TextDocument.URI: edits,
					},
				},
			})
		}
	}

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  out,
	}
}

// handleFoldingRange handles the textDocument/foldingRange method
func (s *LSPServer) handleFoldingRange(request LSPRequest) LSPServerResponse {
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

	ranges := s.foldingRangesForURI(params.TextDocument.URI)
	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  ranges,
	}
}
