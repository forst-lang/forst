package lsp

// handleInitialize handles the initialize method
func (s *LSPServer) handleInitialize(request LSPRequest) LSPServerResponse {
	capabilities := map[string]any{
		"textDocumentSync": map[string]any{
			"openClose": true,
			"change":    1, // Incremental
		},
		"completionProvider": map[string]any{
			"triggerCharacters": []string{".", ":", "(", " "},
			"resolveProvider":   false,
		},
		"hoverProvider": true,
		"diagnosticProvider": map[string]any{
			"identifier": "forst",
		},
		// Standard LSP capabilities for VS Code compatibility
		"definitionProvider": true,
		"referencesProvider": true,
		"documentSymbolProvider": map[string]any{
			"symbolKind": map[string]any{
				"valueSet": []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25},
			},
		},
		"workspaceSymbolProvider":    true,
		"foldingRangeProvider":       true,
		"documentFormattingProvider": true,
		"codeActionProvider": map[string]any{
			"codeActionKinds": []string{"quickfix", "source", "source.formatDocument"},
		},
		"renameProvider": map[string]any{
			"prepareProvider": true,
		},
		"codeLensProvider": map[string]any{
			"resolveProvider": false,
		},
		// Custom LLM debugging capabilities
		"experimental": map[string]any{
			"debugInfoProvider":     true,
			"compilerStateProvider": true,
			"phaseDetailsProvider":  true,
		},
	}

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: map[string]any{
			"capabilities": capabilities,
			"serverInfo": map[string]any{
				"name":    "forst-lsp",
				"version": Version,
			},
		},
	}
}
