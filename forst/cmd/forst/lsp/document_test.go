package lsp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestHandleDidOpen_StoresContentAndReturnsPublishDiagnosticsShape(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("8080", log)
	req := LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/didOpen",
		Params: json.RawMessage(`{
			"textDocument": {
				"uri": "file:///tmp/hover_doc.ft",
				"version": 1,
				"text": "package main\n\nfunc main() {\n}\n"
			}
		}`),
	}
	resp := s.handleDidOpen(req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	pub, ok := resp.Result.(PublishDiagnosticsParams)
	if !ok {
		t.Fatalf("result type %T", resp.Result)
	}
	if pub.URI != "file:///tmp/hover_doc.ft" {
		t.Fatalf("uri = %q", pub.URI)
	}
	if pub.Diagnostics == nil {
		t.Fatal("expected diagnostics slice (may be empty)")
	}

	s.documentMu.RLock()
	text, ok := s.openDocuments["file:///tmp/hover_doc.ft"]
	s.documentMu.RUnlock()
	if !ok {
		t.Fatal("document not stored in openDocuments")
	}
	if !strings.Contains(text, "package main") {
		t.Fatalf("stored content unexpected: %q", text)
	}
}

func TestHandleDidOpen_InvalidJSON_ReturnsParseError(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	resp := s.handleDidOpen(LSPRequest{
		JSONRPC: "2.0",
		ID:      9,
		Params:  json.RawMessage(`not-json`),
	})
	if resp.Error == nil || resp.Error.Code != -32700 {
		t.Fatalf("expected -32700, got %+v", resp.Error)
	}
}
