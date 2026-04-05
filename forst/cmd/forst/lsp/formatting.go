package lsp

import (
	"encoding/json"

	"forst/internal/printer"
)

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

	uri := params.TextDocument.URI
	src, ok := s.openDocumentText(uri)
	if !ok {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result:  nil,
		}
	}

	tabSize := params.Options.TabSize
	if tabSize <= 0 {
		tabSize = 4
	}

	formatted := printer.FormatDocument(src, uri, tabSize, params.Options.InsertSpaces, s.log)
	if formatted == src {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result:  nil,
		}
	}

	end := printer.EndPositionExclusive(src)
	edits := []LSPTextEdit{
		{
			Range: LSPRange{
				Start: LSPPosition{Line: 0, Character: 0},
				End:   LSPPosition{Line: end.Line, Character: end.Character},
			},
			NewText: formatted,
		},
	}
	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  edits,
	}
}

// handleCodeAction handles the textDocument/codeAction method
func (s *LSPServer) handleCodeAction(request LSPRequest) LSPServerResponse {
	// Parse text document and context from params
	var params map[string]interface{}
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

	// For now, return empty array (no code actions available)
	// TODO: Implement actual code actions
	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  []interface{}{},
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
