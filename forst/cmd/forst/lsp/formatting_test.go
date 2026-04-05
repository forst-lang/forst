package lsp

import (
	"encoding/json"
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
