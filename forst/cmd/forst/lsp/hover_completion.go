package lsp

import (
	"encoding/json"
	"runtime"
	"strings"

	"forst/internal/lexer"
)

// handleHover handles the textDocument/hover method
func (s *LSPServer) handleHover(request LSPRequest) LSPServerResponse {
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
			Error: &LSPError{
				Code:    -32700,
				Message: "Parse error",
			},
		}
	}

	hover := s.findHoverForPosition(params.TextDocument.URI, params.Position)

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  hover,
	}
}

// handleCompletion handles the textDocument/completion method
func (s *LSPServer) handleCompletion(request LSPRequest) LSPServerResponse {
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
			Error: &LSPError{
				Code:    -32700,
				Message: "Parse error",
			},
		}
	}

	completions := s.getCompletionsForPosition(params.TextDocument.URI, params.Position)

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: map[string]interface{}{
			"isIncomplete": false,
			"items":        completions,
		},
	}
}

// findHoverForPosition finds hover information for a given position
func (s *LSPServer) findHoverForPosition(uri string, position LSPPosition) *LSPHover {
	// Convert URI to file path
	filePath := strings.TrimPrefix(uri, "file://")
	if runtime.GOOS == "windows" {
		filePath = strings.TrimPrefix(filePath, "/")
	}

	// Only process .ft files
	if !strings.HasSuffix(filePath, ".ft") {
		return nil
	}

	// For now, return basic hover information
	// In a full implementation, this would analyze the AST at the given position
	return &LSPHover{
		Contents: LSPMarkedString{
			Language: "markdown",
			Value:    "Forst language element",
		},
		Range: &LSPRange{
			Start: position,
			End:   position,
		},
	}
}

// getCompletionsForPosition gets completion items for a given position
func (s *LSPServer) getCompletionsForPosition(uri string, position LSPPosition) []LSPCompletionItem {
	// Convert URI to file path
	filePath := strings.TrimPrefix(uri, "file://")
	if runtime.GOOS == "windows" {
		filePath = strings.TrimPrefix(filePath, "/")
	}

	// Only process .ft files
	if !strings.HasSuffix(filePath, ".ft") {
		return nil
	}

	// Return Forst keywords as completion items
	var completions []LSPCompletionItem
	for keyword := range lexer.Keywords {
		completions = append(completions, LSPCompletionItem{
			Label:            keyword,
			Kind:             LSPCompletionItemKindKeyword,
			Detail:           "Forst keyword",
			Documentation:    keyword,
			InsertText:       keyword,
			InsertTextFormat: LSPInsertTextFormatPlainText,
		})
	}

	return completions
}

// handleWorkspaceSymbol handles the workspace/symbol method
func (s *LSPServer) handleWorkspaceSymbol(request LSPRequest) LSPServerResponse {
	// Parse query from params
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

	// For now, return empty array (no workspace symbols found)
	// TODO: Implement actual workspace symbol search
	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  []interface{}{},
	}
}
