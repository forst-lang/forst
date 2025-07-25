package lsp

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

// handleDidOpen handles the textDocument/didOpen method
func (s *LSPServer) handleDidOpen(request LSPRequest) LSPServerResponse {
	var params struct {
		TextDocument struct {
			URI     string `json:"uri"`
			Version int    `json:"version"`
			Text    string `json:"text"`
		} `json:"textDocument"`
	}

	if err := json.Unmarshal(request.Params, &params); err != nil {
		s.log.Errorf("Failed to parse didOpen params: %v", err)
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &LSPError{
				Code:    -32700,
				Message: "Parse error",
			},
		}
	}

	s.log.WithFields(logrus.Fields{
		"uri":            params.TextDocument.URI,
		"version":        params.TextDocument.Version,
		"content_length": len(params.TextDocument.Text),
	}).Info("File opened for compilation")

	// Process the Forst file and get diagnostics
	diagnostics := s.processForstFile(params.TextDocument.URI, params.TextDocument.Text)

	// Send diagnostics notification
	s.sendDiagnosticsNotification(params.TextDocument.URI, diagnostics)

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  nil,
	}
}

// handleDidChange handles the textDocument/didChange method
func (s *LSPServer) handleDidChange(request LSPRequest) LSPServerResponse {
	var params struct {
		TextDocument struct {
			URI     string `json:"uri"`
			Version int    `json:"version"`
		} `json:"textDocument"`
		ContentChanges []struct {
			Text string `json:"text"`
		} `json:"contentChanges"`
	}

	if err := json.Unmarshal(request.Params, &params); err != nil {
		s.log.Errorf("Failed to parse didChange params: %v", err)
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &LSPError{
				Code:    -32700,
				Message: "Parse error",
			},
		}
	}

	// Get the latest content from the changes
	var latestContent string
	if len(params.ContentChanges) > 0 {
		latestContent = params.ContentChanges[len(params.ContentChanges)-1].Text
	}

	s.log.WithFields(logrus.Fields{
		"uri":            params.TextDocument.URI,
		"version":        params.TextDocument.Version,
		"changes_count":  len(params.ContentChanges),
		"content_length": len(latestContent),
	}).Info("File content changed")

	// Process the Forst file and get diagnostics
	diagnostics := s.processForstFile(params.TextDocument.URI, latestContent)

	// Send diagnostics notification
	s.sendDiagnosticsNotification(params.TextDocument.URI, diagnostics)

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  nil,
	}
}

// handleDidClose handles the textDocument/didClose method
func (s *LSPServer) handleDidClose(request LSPRequest) LSPServerResponse {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}

	if err := json.Unmarshal(request.Params, &params); err != nil {
		s.log.Errorf("Failed to parse didClose params: %v", err)
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &LSPError{
				Code:    -32700,
				Message: "Parse error",
			},
		}
	}

	s.log.WithFields(logrus.Fields{
		"uri": params.TextDocument.URI,
	}).Info("File closed")

	// Clear diagnostics for the closed document
	s.sendDiagnosticsNotification(params.TextDocument.URI, []LSPDiagnostic{})

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  nil,
	}
}

// processForstFile processes a Forst file and returns diagnostics
func (s *LSPServer) processForstFile(uri, content string) []LSPDiagnostic {
	// Convert URI to file path
	filePath := strings.TrimPrefix(uri, "file://")
	if runtime.GOOS == "windows" {
		filePath = strings.TrimPrefix(filePath, "/")
	}

	// Only process .ft files
	if !strings.HasSuffix(filePath, ".ft") {
		return nil
	}

	// Create a temporary file for compilation
	tempFile, err := os.CreateTemp("", "forst-lsp-*.ft")
	if err != nil {
		s.log.Errorf("Failed to create temp file: %v", err)
		return []LSPDiagnostic{
			{
				Range: LSPRange{
					Start: LSPPosition{Line: 0, Character: 0},
					End:   LSPPosition{Line: 0, Character: 0},
				},
				Severity: LSPDiagnosticSeverityError,
				Message:  fmt.Sprintf("Failed to create temp file: %v", err),
			},
		}
	}
	defer os.Remove(tempFile.Name())

	// Write content to temp file
	if _, err := tempFile.WriteString(content); err != nil {
		s.log.Errorf("Failed to write to temp file: %v", err)
		return []LSPDiagnostic{
			{
				Range: LSPRange{
					Start: LSPPosition{Line: 0, Character: 0},
					End:   LSPPosition{Line: 0, Character: 0},
				},
				Severity: LSPDiagnosticSeverityError,
				Message:  fmt.Sprintf("Failed to write to temp file: %v", err),
			},
		}
	}
	tempFile.Close()

	// Compile the file and get diagnostics
	debugger := s.debugger.GetDebugger(PhaseParser, tempFile.Name())
	return s.compileForstFile(tempFile.Name(), content, debugger)
}

// handleDefinition handles the textDocument/definition method
func (s *LSPServer) handleDefinition(request LSPRequest) LSPServerResponse {
	// Parse position from params
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

	// For now, return null (no definition found)
	// TODO: Implement actual definition lookup
	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  nil,
	}
}

// handleReferences handles the textDocument/references method
func (s *LSPServer) handleReferences(request LSPRequest) LSPServerResponse {
	// Parse position from params
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

	// For now, return empty array (no references found)
	// TODO: Implement actual reference lookup
	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  []interface{}{},
	}
}

// handleDocumentSymbol handles the textDocument/documentSymbol method
func (s *LSPServer) handleDocumentSymbol(request LSPRequest) LSPServerResponse {
	// Parse text document from params
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

	// For now, return empty array (no symbols found)
	// TODO: Implement actual symbol extraction
	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  []interface{}{},
	}
}
