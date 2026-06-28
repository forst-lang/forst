package lsp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestHandleCodeAction_providersExample_noPanic(t *testing.T) {
	t.Parallel()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "examples", "in", "rfc", "providers", "providers.ft"))
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(root)
	if err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, root)
	s := NewLSPServer("8080", logrus.New())
	s.setOpenDocument(uri, string(src))

	params, err := json.Marshal(map[string]any{
		"textDocument": map[string]string{"uri": uri},
		"range": map[string]any{
			"start": map[string]any{"line": 7, "character": 0},
			"end":   map[string]any{"line": 10, "character": 1},
		},
		"context": map[string]any{
			"only": []string{"source", "quickfix"},
		},
		"diagnostics": []map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("handleCodeAction panicked: %v", r)
		}
	}()

	resp := s.handleCodeAction(LSPRequest{
		JSONRPC: "2.0",
		ID:      99,
		Params:  json.RawMessage(params),
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	// Ensure response marshals (HTTP handler does this).
	if _, err := json.Marshal(resp); err != nil {
		t.Fatalf("marshal response: %v", err)
	}
}

func TestHandleLSP_httpCodeAction_providersExample_returns200(t *testing.T) {
	t.Parallel()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "examples", "in", "rfc", "providers", "providers.ft"))
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(root)
	if err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, root)
	s := NewLSPServer("8080", logrus.New())
	s.setOpenDocument(uri, string(src))

	openParams, err := json.Marshal(map[string]any{
		"textDocument": map[string]any{
			"uri":        uri,
			"languageId": "forst",
			"version":    1,
			"text":       string(src),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	actionParams, err := json.Marshal(map[string]any{
		"textDocument": map[string]string{"uri": uri},
		"range": map[string]any{
			"start": map[string]any{"line": 7, "character": 0},
			"end":   map[string]any{"line": 10, "character": 1},
		},
		"context": map[string]any{
			"diagnostics": []any{},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	post := func(id int, method string, params []byte) *httptest.ResponseRecorder {
		body, err := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"method":  method,
			"params":  json.RawMessage(params),
		})
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		w := httptest.NewRecorder()
		s.handleLSP(w, req)
		return w
	}

	openW := post(1, "textDocument/didOpen", openParams)
	if openW.Code != http.StatusOK {
		t.Fatalf("didOpen status %d body %s", openW.Code, openW.Body.String())
	}

	actionW := post(2, "textDocument/codeAction", actionParams)
	if actionW.Code != http.StatusOK {
		t.Fatalf("codeAction status %d body %s", actionW.Code, actionW.Body.String())
	}
	var parsed LSPServerResponse
	if err := json.Unmarshal(actionW.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal: %v body %s", err, actionW.Body.String())
	}
	if parsed.Error != nil {
		t.Fatalf("unexpected error: %+v", parsed.Error)
	}
}
