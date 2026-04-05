package lsp

import (
	"encoding/json"
	"path/filepath"
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

func TestHandleDidChange_InvalidJSON_ReturnsParseError(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	resp := s.handleDidChange(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  json.RawMessage(`{`),
	})
	if resp.Error == nil || resp.Error.Code != -32700 {
		t.Fatalf("expected -32700, got %+v", resp.Error)
	}
}

func TestHandleDidChange_UsesLastContentChange(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	uri := "file:///tmp/didchange.ft"
	first := "package main\n"
	second := "package main\n\nfunc main() {}\n"
	raw, err := json.Marshal(map[string]interface{}{
		"textDocument": map[string]interface{}{"uri": uri, "version": 2},
		"contentChanges": []map[string]string{
			{"text": first},
			{"text": second},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	req := LSPRequest{
		JSONRPC: "2.0",
		ID:      2,
		Params:  json.RawMessage(raw),
	}
	resp := s.handleDidChange(req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	s.documentMu.RLock()
	got, ok := s.openDocuments[uri]
	s.documentMu.RUnlock()
	if !ok {
		t.Fatal("document not stored")
	}
	if got != second {
		t.Fatalf("want last change text, got %q", got)
	}
}

func TestHandleDidClose_InvalidJSON_ReturnsParseError(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	resp := s.handleDidClose(LSPRequest{
		JSONRPC: "2.0",
		ID:      3,
		Params:  json.RawMessage(`not-json`),
	})
	if resp.Error == nil || resp.Error.Code != -32700 {
		t.Fatalf("expected -32700, got %+v", resp.Error)
	}
}

func TestHandleDidClose_RemovesOpenDocument(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	uri := "file:///tmp/closeme.ft"
	s.documentMu.Lock()
	s.openDocuments[uri] = "package main\n"
	s.documentMu.Unlock()

	closeRaw, err := json.Marshal(map[string]interface{}{
		"textDocument": map[string]string{"uri": uri},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp := s.handleDidClose(LSPRequest{
		JSONRPC: "2.0",
		ID:      4,
		Params:  json.RawMessage(closeRaw),
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	s.documentMu.RLock()
	_, still := s.openDocuments[uri]
	s.documentMu.RUnlock()
	if still {
		t.Fatal("expected uri removed from openDocuments")
	}
	pub, ok := resp.Result.(PublishDiagnosticsParams)
	if !ok || len(pub.Diagnostics) != 0 {
		t.Fatalf("expected empty diagnostics, got %#v", resp.Result)
	}
}

func TestProcessForstFile_nonFtPathReturnsNil(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	if d := s.processForstFile("file:///proj/main.go", "package main\n"); d != nil {
		t.Fatalf("expected nil for .go, got %d diags", len(d))
	}
}

func TestProcessForstFile_ftPathParseErrorReturnsDiagnostics(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	ft := filepath.Join(t.TempDir(), "bad.ft")
	uri := "file://" + ft
	d := s.processForstFile(uri, "package main\n\n!!!\n")
	if len(d) == 0 {
		t.Fatal("expected at least one diagnostic for invalid syntax")
	}
}
