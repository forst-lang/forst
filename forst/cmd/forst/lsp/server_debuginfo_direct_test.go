package lsp

import (
	"encoding/json"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestHandleDebugInfoDirect_invalidParams(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	req := LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "forst/debugInfo",
		Params:  json.RawMessage(`{bad json`),
	}
	resp := s.HandleDebugInfoDirect(req, "package main\n")
	if resp.Error == nil {
		t.Fatal("expected parse params error")
	}
}

func TestCreateCompressedDebugData_containsPayloadAndRatio(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	out := s.createCompressedDebugData(map[string]interface{}{"a": 1, "b": "x"})
	if out["encoding"] != "gzip+base64" {
		t.Fatalf("encoding: %v", out["encoding"])
	}
	if out["data"] == "" {
		t.Fatal("expected compressed data")
	}
	if _, ok := out["compression_ratio"].(float64); !ok {
		t.Fatalf("compression_ratio type: %T", out["compression_ratio"])
	}
}

func TestGetPhaseSummaries_includesAllPhases(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	sum := s.getPhaseSummaries("file:///tmp/a.ft")
	for _, key := range []string{"lexer", "parser", "typechecker", "transformer"} {
		if _, ok := sum[key]; !ok {
			t.Fatalf("missing phase summary key %s", key)
		}
	}
}

