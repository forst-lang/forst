package lsp

import (
	"encoding/json"
	"path/filepath"
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

	s.documentMu.Lock()
	s.openDocuments[params.TextDocument.URI] = params.TextDocument.Text
	s.documentMu.Unlock()

	memberURIs := s.samePackageOpenURIs(params.TextDocument.URI)
	diagnostics := s.processForstFileWithURIs(params.TextDocument.URI, params.TextDocument.Text, memberURIs)
	s.sendDiagnosticsNotification(params.TextDocument.URI, diagnostics)
	s.publishPeerDiagnosticsFromGroup(memberURIs, params.TextDocument.URI)

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: PublishDiagnosticsParams{
			URI:         params.TextDocument.URI,
			Diagnostics: diagnostics,
		},
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

	s.documentMu.Lock()
	s.openDocuments[params.TextDocument.URI] = latestContent
	s.documentMu.Unlock()

	memberURIs := s.samePackageOpenURIs(params.TextDocument.URI)
	diagnostics := s.processForstFileWithURIs(params.TextDocument.URI, latestContent, memberURIs)
	s.sendDiagnosticsNotification(params.TextDocument.URI, diagnostics)
	s.publishPeerDiagnosticsFromGroup(memberURIs, params.TextDocument.URI)

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: PublishDiagnosticsParams{
			URI:         params.TextDocument.URI,
			Diagnostics: diagnostics,
		},
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

	s.documentMu.Lock()
	delete(s.openDocuments, params.TextDocument.URI)
	s.documentMu.Unlock()

	s.invalidatePeerAnalysisCache(params.TextDocument.URI)
	s.invalidatePackageAnalysisCache()

	// Remaining open buffers in the same directory may switch from merged to single-file analysis.
	s.republishOpenFtInSameDir(params.TextDocument.URI)

	// Clear diagnostics for the closed document
	s.sendDiagnosticsNotification(params.TextDocument.URI, []LSPDiagnostic{})

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: PublishDiagnosticsParams{
			URI:         params.TextDocument.URI,
			Diagnostics: []LSPDiagnostic{},
		},
	}
}

// processForstFile processes a Forst file and returns diagnostics.
func (s *LSPServer) processForstFile(uri, content string) []LSPDiagnostic {
	return s.processForstFileWithURIs(uri, content, nil)
}

// processForstFileWithURIs is like processForstFile but accepts an optional precomputed same-package
// member list (from samePackageOpenURIs) to avoid redundant work on the hot path.
func (s *LSPServer) processForstFileWithURIs(uri, content string, memberURIs []string) []LSPDiagnostic {
	filePath := filePathFromDocumentURI(uri)
	if !strings.HasSuffix(filePath, ".ft") {
		return nil
	}

	uris := memberURIs
	if uris == nil {
		uris = s.samePackageOpenURIs(uri)
	}
	if len(uris) > 1 {
		return s.compileForstFilePackageGroup(uri, filePath, content, uris)
	}
	debugger := s.debugger.GetDebugger(PhaseParser, filePath)
	return s.compileForstFile(filePath, content, debugger)
}

// republishOpenFtInSameDir re-runs diagnostics for other open .ft files in the same directory
// (e.g. after one peer closes, survivors may no longer use merged package analysis).
func (s *LSPServer) republishOpenFtInSameDir(closedURI string) {
	dir := filepath.Dir(filePathFromDocumentURI(closedURI))
	s.documentMu.RLock()
	var peers []string
	for u := range s.openDocuments {
		if u == closedURI {
			continue
		}
		if !isForstDocumentURI(u) {
			continue
		}
		if filepath.Dir(filePathFromDocumentURI(u)) != dir {
			continue
		}
		peers = append(peers, u)
	}
	s.documentMu.RUnlock()
	for _, u := range peers {
		s.documentMu.RLock()
		c := s.openDocuments[u]
		s.documentMu.RUnlock()
		if c == "" {
			continue
		}
		memberURIs := s.samePackageOpenURIs(u)
		d := s.processForstFileWithURIs(u, c, memberURIs)
		s.sendDiagnosticsNotification(u, d)
	}
}

// publishPeerDiagnosticsFromGroup republishes diagnostics for other open buffers in the same package group.
// memberURIs must be the result of samePackageOpenURIs for exceptURI (same ordering as fingerprinting).
func (s *LSPServer) publishPeerDiagnosticsFromGroup(memberURIs []string, exceptURI string) {
	if len(memberURIs) <= 1 {
		return
	}
	for _, peer := range memberURIs {
		if peer == exceptURI {
			continue
		}
		s.documentMu.RLock()
		c := s.openDocuments[peer]
		s.documentMu.RUnlock()
		if c == "" {
			continue
		}
		d := s.processForstFileWithURIs(peer, c, memberURIs)
		s.sendDiagnosticsNotification(peer, d)
	}
}
