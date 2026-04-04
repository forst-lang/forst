package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestHandleFormatting_InvalidParams_ReturnsInvalidParams(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	resp := s.handleFormatting(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  json.RawMessage(`{`),
	})
	if resp.Error == nil {
		t.Fatal("expected error")
	}
	if resp.Error.Code != -32602 {
		t.Fatalf("code = %d", resp.Error.Code)
	}
}

func TestHandleFormatting_ValidEmptyParams_ReturnsNilResult(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	resp := s.handleFormatting(LSPRequest{
		JSONRPC: "2.0",
		ID:      2,
		Params:  json.RawMessage(`{}`),
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	if resp.Result != nil {
		t.Fatalf("expected nil formatting result, got %v", resp.Result)
	}
}

func TestHandleCodeAction_InvalidParams_ReturnsInvalidParams(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	resp := s.handleCodeAction(LSPRequest{
		JSONRPC: "2.0",
		ID:      3,
		Params:  json.RawMessage(`not-json`),
	})
	if resp.Error == nil || resp.Error.Code != -32602 {
		t.Fatalf("got %+v", resp.Error)
	}
}

func TestHandleCodeLens_ReturnsEmptySlice(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	resp := s.handleCodeLens(LSPRequest{
		JSONRPC: "2.0",
		ID:      4,
		Params:  json.RawMessage(`{"textDocument": {"uri": "file:///x.ft"}}`),
	})
	if resp.Error != nil {
		t.Fatal(resp.Error)
	}
	arr, ok := resp.Result.([]interface{})
	if !ok || len(arr) != 0 {
		t.Fatalf("expected empty array, got %T %v", resp.Result, resp.Result)
	}
}

func TestHandleFoldingRange_InvalidParams_ReturnsInvalidParams(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	resp := s.handleFoldingRange(LSPRequest{
		JSONRPC: "2.0",
		ID:      5,
		Params:  json.RawMessage(`{`),
	})
	if resp.Error == nil || resp.Error.Code != -32602 {
		t.Fatalf("got %+v", resp.Error)
	}
}

func TestHandleFoldingRange_parseOk_returnsFunctionBodyRegion(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module foldt\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ft := filepath.Join(dir, "fold.ft")
	const src = "package main\n\nfunc main() {\n  var x: Int = 1\n}\n"
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := "file://" + ft
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	params, err := json.Marshal(map[string]interface{}{
		"textDocument": map[string]string{"uri": uri},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp := s.handleFoldingRange(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  json.RawMessage(params),
	})
	if resp.Error != nil {
		t.Fatal(resp.Error)
	}
	arr, ok := resp.Result.([]interface{})
	if !ok || len(arr) == 0 {
		t.Fatalf("expected folding ranges, got %T %#v", resp.Result, resp.Result)
	}
	m, ok := arr[0].(map[string]interface{})
	if !ok {
		t.Fatalf("range type %T", arr[0])
	}
	sl := m["startLine"]
	if sl != float64(2) && sl != 2 { // int from in-process maps; float64 after JSON round-trip
		t.Fatalf("startLine = %v (%T)", sl, sl)
	}
	if m["kind"] != "region" {
		t.Fatalf("kind = %v", m["kind"])
	}
}

func TestFoldingRangesForURI_parseError_returnsEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "bad.ft")
	uri := "file://" + ft
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = "package main\n\nunexpected\n"
	s.documentMu.Unlock()

	if err := os.WriteFile(ft, []byte("package main\n\nunexpected\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := s.foldingRangesForURI(uri)
	if len(got) != 0 {
		t.Fatalf("expected no ranges on parse error, got %d", len(got))
	}
}
