package lsp

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"forst/internal/printer"

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

func TestHandleFormatting_EmptyParams_ReturnsInvalidParams(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	resp := s.handleFormatting(LSPRequest{
		JSONRPC: "2.0",
		ID:      2,
		Params:  json.RawMessage(`{}`),
	})
	if resp.Error == nil || resp.Error.Code != -32602 {
		t.Fatalf("expected invalid params, got %+v", resp.Error)
	}
}

func TestHandleFormatting_UnknownDocument_ReturnsNil(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	uri := mustFileURI(t, filepath.Join(t.TempDir(), "not-open.ft"))
	params, err := json.Marshal(map[string]interface{}{
		"textDocument": map[string]string{"uri": uri},
		"options":      map[string]interface{}{"tabSize": 4, "insertSpaces": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp := s.handleFormatting(LSPRequest{
		JSONRPC: "2.0",
		ID:      21,
		Params:  json.RawMessage(params),
	})
	if resp.Error != nil {
		t.Fatal(resp.Error)
	}
	if resp.Result != nil {
		t.Fatalf("expected nil, got %v", resp.Result)
	}
}

func TestHandleFormatting_TrimsTrailingWhitespace_ReturnsSingleEdit(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	s := NewLSPServer("8080", logrus.New())
	uri := mustFileURI(t, filepath.Join(t.TempDir(), "fmt.ft"))
	src := "package main  \nfunc main() {\n}\n"
	s.setOpenDocument(uri, src)
	params, err := json.Marshal(map[string]interface{}{
		"textDocument": map[string]string{"uri": uri},
		"options":      map[string]interface{}{"tabSize": 4, "insertSpaces": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp := s.handleFormatting(LSPRequest{
		JSONRPC: "2.0",
		ID:      22,
		Params:  json.RawMessage(params),
	})
	if resp.Error != nil {
		t.Fatal(resp.Error)
	}
	edits, ok := resp.Result.([]LSPTextEdit)
	if !ok || len(edits) != 1 {
		t.Fatalf("expected one edit, got %T %#v", resp.Result, resp.Result)
	}
	pretty, err := printer.FormatSource(src, "fmt.ft", log)
	if err != nil {
		t.Fatal(err)
	}
	want := printer.FormatForstWhitespace(pretty, 4, true)
	if edits[0].NewText != want {
		t.Fatalf("newText = %q, want %q", edits[0].NewText, want)
	}
	if edits[0].Range.Start.Line != 0 || edits[0].Range.Start.Character != 0 {
		t.Fatalf("range start = %+v", edits[0].Range.Start)
	}
}

func TestHandleFormatting_AlreadyFormatted_ReturnsNil(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	initial := "package main\n\nfunc main() {\n}\n"
	pretty, err := printer.FormatSource(initial, "ok.ft", log)
	if err != nil {
		t.Fatal(err)
	}
	src := printer.FormatForstWhitespace(pretty, 4, true)
	pretty2, err := printer.FormatSource(src, "ok.ft", log)
	if err != nil {
		t.Fatal(err)
	}
	if printer.FormatForstWhitespace(pretty2, 4, true) != src {
		t.Fatalf("pretty-print is not stable under whitespace pass")
	}

	s := NewLSPServer("8080", logrus.New())
	uri := mustFileURI(t, filepath.Join(t.TempDir(), "ok.ft"))
	s.setOpenDocument(uri, src)
	params, err := json.Marshal(map[string]interface{}{
		"textDocument": map[string]string{"uri": uri},
		"options":      map[string]interface{}{"tabSize": 4, "insertSpaces": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp := s.handleFormatting(LSPRequest{
		JSONRPC: "2.0",
		ID:      23,
		Params:  json.RawMessage(params),
	})
	if resp.Error != nil {
		t.Fatal(resp.Error)
	}
	if resp.Result != nil {
		t.Fatalf("expected nil when unchanged, got %v", resp.Result)
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
	uri := mustFileURI(t, filepath.Join(t.TempDir(), "x.ft"))
	params, err := json.Marshal(map[string]interface{}{
		"textDocument": map[string]string{"uri": uri},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp := s.handleCodeLens(LSPRequest{
		JSONRPC: "2.0",
		ID:      4,
		Params:  json.RawMessage(params),
	})
	if resp.Error != nil {
		t.Fatal(resp.Error)
	}
	arr, ok := resp.Result.([]interface{})
	if !ok || len(arr) != 0 {
		t.Fatalf("expected empty array, got %T %v", resp.Result, resp.Result)
	}
}
