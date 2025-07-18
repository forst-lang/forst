package lsp

// handleShutdown handles the shutdown method
func (s *LSPServer) handleShutdown(request LSPRequest) LSPServerResponse {
	// Perform cleanup operations here
	s.log.Info("LSP server shutdown requested")

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  nil,
	}
}

// handleExit handles the exit method
func (s *LSPServer) handleExit(request LSPRequest) LSPServerResponse {
	// Perform final cleanup and exit
	s.log.Info("LSP server exit requested")

	// Stop the server
	if err := s.Stop(); err != nil {
		s.log.Errorf("Error stopping server: %v", err)
	}

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  nil,
	}
}
