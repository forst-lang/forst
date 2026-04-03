package lsp

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestHandleShutdown_ReturnsNilResult(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	resp := s.handleShutdown(LSPRequest{
		JSONRPC: "2.0",
		ID:      7,
		Method:  "shutdown",
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	if resp.Result != nil {
		t.Fatalf("expected nil result, got %v", resp.Result)
	}
	if resp.ID != 7 {
		t.Fatalf("id = %v", resp.ID)
	}
}

func TestHandleExit_WithNoHTTPListener_StillReturnsNilResult(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	// s.server is nil until Start(); Stop is a no-op
	resp := s.handleExit(LSPRequest{
		JSONRPC: "2.0",
		ID:      8,
		Method:  "exit",
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	if resp.Result != nil {
		t.Fatalf("expected nil result, got %v", resp.Result)
	}
}
