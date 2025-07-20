package lsp

// handleInitialize handles the initialize method
func (s *LSPServer) handleInitialize(request LSPRequest) LSPServerResponse {
	capabilities := map[string]interface{}{
		"textDocumentSync": map[string]interface{}{
			"openClose": true,
			"change":    1, // Incremental
		},
		"completionProvider": map[string]interface{}{
			"triggerCharacters": []string{".", ":", "(", " "},
			"resolveProvider":   true,
		},
		"hoverProvider": true,
		"diagnosticProvider": map[string]interface{}{
			"identifier": "forst",
		},
		// Standard LSP capabilities for VS Code compatibility
		"definitionProvider": true,
		"referencesProvider": true,
		"documentSymbolProvider": map[string]interface{}{
			"symbolKind": map[string]interface{}{
				"valueSet": []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25},
			},
		},
		"workspaceSymbolProvider":    true,
		"documentFormattingProvider": true,
		"codeActionProvider": map[string]interface{}{
			"codeActionKinds": []string{"quickfix", "refactor", "source"},
		},
		"codeLensProvider": map[string]interface{}{
			"resolveProvider": true,
		},
		"foldingRangeProvider": true,
		// Custom LLM debugging capabilities
		"experimental": map[string]interface{}{
			"debugInfoProvider":     true,
			"compilerStateProvider": true,
			"phaseDetailsProvider":  true,
		},
	}

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: map[string]interface{}{
			"capabilities": capabilities,
			"serverInfo": map[string]interface{}{
				"name":    "forst-lsp",
				"version": Version,
			},
		},
	}
}
