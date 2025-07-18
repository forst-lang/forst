package lsp

// handleInitialize handles the initialize method
func (s *LSPServer) handleInitialize(request LSPRequest) LSPServerResponse {
	capabilities := map[string]interface{}{
		"textDocumentSync": map[string]interface{}{
			"openClose": true,
			"change":    1, // Incremental
		},
		"completionProvider": map[string]interface{}{
			"triggerCharacters": []string{".", ":", "("},
		},
		"hoverProvider": true,
		"diagnosticProvider": map[string]interface{}{
			"identifier": "forst",
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
