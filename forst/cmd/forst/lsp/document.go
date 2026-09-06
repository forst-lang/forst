package lsp

import (
	"encoding/json"
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
	}).Debug("File opened for compilation")

	s.setOpenDocument(params.TextDocument.URI, params.TextDocument.Text)

	memberURIs := s.samePackageOpenURIs(params.TextDocument.URI)
	diagnostics := s.processForstFileWithURIs(params.TextDocument.URI, params.TextDocument.Text)
	s.sendDiagnosticsNotification(params.TextDocument.URI, diagnostics)
	s.publishPeerDiagnosticsFromGroup(memberURIs, params.TextDocument.URI, diagnostics)

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
	}).Debug("File content changed")

	s.setOpenDocument(params.TextDocument.URI, latestContent)

	memberURIs := s.samePackageOpenURIs(params.TextDocument.URI)
	diagnostics := s.processForstFileWithURIs(params.TextDocument.URI, latestContent)
	s.sendDiagnosticsNotification(params.TextDocument.URI, diagnostics)
	s.publishPeerDiagnosticsFromGroup(memberURIs, params.TextDocument.URI, diagnostics)

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
	}).Debug("File closed")

	s.deleteOpenDocument(params.TextDocument.URI)

	s.invalidatePeerAnalysisCache(params.TextDocument.URI)
	s.removePackageSnapshotsReferencingURI(params.TextDocument.URI)
	s.invalidateFileParseCache(params.TextDocument.URI)

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
	return s.processForstFileWithURIs(uri, content)
}

// processForstFileWithURIs is like processForstFile; membership for merged package analysis is always
// derived from samePackageOpenURIs(uri) so disk peers and open buffers stay a single source of truth.
func (s *LSPServer) processForstFileWithURIs(uri, content string) []LSPDiagnostic {
	s.processForstFileInvocations.Add(1)
	filePath := filePathFromDocumentURI(uri)
	if filePath == "" || !strings.HasSuffix(strings.ToLower(filePath), ".ft") {
		return nil
	}

	uris := s.samePackageOpenURIs(uri)
	if len(uris) > 1 {
		return s.compileForstFilePackageGroup(uri, filePath, content)
	}
	debugger := s.debugger.GetDebugger(PhaseParser, filePath)
	return s.compileForstFile(filePath, content, debugger)
}

// republishOpenFtInSameDir re-runs diagnostics for other open .ft files in the same directory
// (e.g. after one peer closes, survivors may no longer use merged package analysis).
func (s *LSPServer) republishOpenFtInSameDir(closedURI string) {
	closedPath := filePathFromDocumentURI(closedURI)
	if closedPath == "" {
		return
	}
	dir := canonicalDirForPath(closedPath)
	canonClosed := canonicalFileURI(closedURI)
	s.documentMu.RLock()
	seen := make(map[string]struct{}, len(s.openDocuments))
	var peers []string
	for u := range s.openDocuments {
		cu := canonicalFileURI(u)
		if _, ok := seen[cu]; ok {
			continue
		}
		seen[cu] = struct{}{}
		if cu == canonClosed {
			continue
		}
		if !isForstDocumentURI(cu) {
			continue
		}
		p := filePathFromDocumentURI(cu)
		if p == "" {
			continue
		}
		if canonicalDirForPath(p) != dir {
			continue
		}
		peers = append(peers, cu)
	}
	s.documentMu.RUnlock()
	for _, u := range peers {
		c := s.openDocumentContent(u)
		if c == "" {
			continue
		}
		d := s.processForstFileWithURIs(u, c)
		s.sendDiagnosticsNotification(u, d)
	}
}

// publishPeerDiagnosticsFromGroup republishes diagnostics for other open buffers in the same package group.
// memberURIs must be the result of samePackageOpenURIs for exceptURI (same ordering as fingerprinting).
// shared is the diagnostics already computed for exceptURI from one package-group analyze+transform;
// peers reuse that work instead of re-running processForstFileWithURIs (which would re-transform).
func (s *LSPServer) publishPeerDiagnosticsFromGroup(memberURIs []string, exceptURI string, shared []LSPDiagnostic) {
	if len(memberURIs) <= 1 {
		return
	}
	exceptCanon := canonicalFileURI(exceptURI)
	if shared == nil {
		shared = []LSPDiagnostic{}
	}
	for _, peer := range memberURIs {
		if canonicalFileURI(peer) == exceptCanon {
			continue
		}
		// Disk-only peers are not open; skip. Open buffers get the shared package-group result.
		if c := s.openDocumentContent(peer); c == "" {
			continue
		}
		s.sendDiagnosticsNotification(peer, shared)
	}
}

// setOpenDocument stores buffer text under the canonical URI and, when different, the raw URI
// so lookups succeed whether the client uses normalized file:// spelling or not.
func (s *LSPServer) setOpenDocument(uri, text string) {
	canon := canonicalFileURI(uri)
	s.documentMu.Lock()
	s.openDocuments[canon] = text
	if canon != uri {
		s.openDocuments[uri] = text
	}
	s.documentMu.Unlock()
}

// deleteOpenDocument removes both canonical and alias keys for uri.
func (s *LSPServer) deleteOpenDocument(uri string) {
	canon := canonicalFileURI(uri)
	s.documentMu.Lock()
	delete(s.openDocuments, canon)
	delete(s.openDocuments, uri)
	s.documentMu.Unlock()
}

// openDocumentContent returns the latest open buffer text for uri, trying canonical and raw keys.
func (s *LSPServer) openDocumentContent(uri string) string {
	s.documentMu.RLock()
	defer s.documentMu.RUnlock()
	canon := canonicalFileURI(uri)
	if c := s.openDocuments[canon]; c != "" {
		return c
	}
	return s.openDocuments[uri]
}

// openDocumentText returns the open buffer for uri and whether that URI is tracked (including empty buffers).
func (s *LSPServer) openDocumentText(uri string) (string, bool) {
	s.documentMu.RLock()
	defer s.documentMu.RUnlock()
	canon := canonicalFileURI(uri)
	if _, ok := s.openDocuments[canon]; ok {
		return s.openDocuments[canon], true
	}
	if _, ok := s.openDocuments[uri]; ok {
		return s.openDocuments[uri], true
	}
	return "", false
}

func (s *LSPServer) removePackageSnapshotsReferencingURI(uri string) {
	s.packageAnalysis.removeSnapshotReferencingURI(uri)
}

func (s *LSPServer) invalidateFileParseCache(uri string) {
	if s.fileParseCache != nil {
		s.fileParseCache.remove(uri)
	}
}
