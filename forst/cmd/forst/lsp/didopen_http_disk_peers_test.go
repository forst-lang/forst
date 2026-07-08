package lsp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"forst/internal/testutil"

	"github.com/sirupsen/logrus"
)

func postLSPMethod(t *testing.T, s *LSPServer, id int, method string, params any) LSPServerResponse {
	t.Helper()
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  json.RawMessage(paramsJSON),
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.handleLSP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("%s status %d body %s", method, w.Code, w.Body.String())
	}
	var resp LSPServerResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal %s response: %v body %s", method, err, w.Body.String())
	}
	if resp.Error != nil {
		t.Fatalf("%s error: %+v", method, resp.Error)
	}
	return resp
}

func diagnosticsFromDidOpenResult(t *testing.T, resp LSPServerResponse) []LSPDiagnostic {
	t.Helper()
	pub, ok := resp.Result.(PublishDiagnosticsParams)
	if !ok {
		// handleLSP unmarshals through JSON; re-decode when Result is map[string]any.
		raw, err := json.Marshal(resp.Result)
		if err != nil {
			t.Fatalf("marshal result: %v", err)
		}
		if err := json.Unmarshal(raw, &pub); err != nil {
			t.Fatalf("result type %T: %v", resp.Result, err)
		}
	}
	return pub.Diagnostics
}

func TestHandleLSP_didOpen_probeExec_diskPeers_noGoImport(t *testing.T) {
	t.Parallel()
	root, execPath := testutil.WriteProbeModuleFixture(t, true)
	execSrc, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatal(err)
	}
	execURI := mustFileURI(t, execPath)

	s := NewLSPServer("8080", logrus.New())
	rootURI := mustFileURI(t, root)
	postLSPMethod(t, s, 1, "initialize", map[string]any{
		"processId": nil,
		"rootUri":   rootURI,
		"capabilities": map[string]any{},
	})

	resp := postLSPMethod(t, s, 2, "textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        execURI,
			"languageId": "forst",
			"version":    1,
			"text":       string(execSrc),
		},
	})
	assertNoGoImportDiagnostics(t, diagnosticsFromDidOpenResult(t, resp))
}

func TestHandleLSP_didOpen_probeExec_manyDiskPeers_noGoImport(t *testing.T) {
	t.Parallel()
	root, execPath := testutil.WriteProbeModuleFixture(t, true)
	probeDir := filepath.Join(root, "internal", "probe")
	peers := []struct {
		name    string
		content string
	}{
		{"helper_a.ft", "package jobs\n\nfunc helperA(): Int { return 1 }\n"},
		{"helper_b.ft", "package jobs\n\nfunc helperB(): Int { return 2 }\n"},
		{"helper_c.ft", "package jobs\n\nfunc helperC(): Int { return 3 }\n"},
		{"helper_d.ft", "package jobs\n\nfunc helperD(): Int { return 4 }\n"},
	}
	for _, p := range peers {
		if err := os.WriteFile(filepath.Join(probeDir, p.name), []byte(p.content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	execSrc, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatal(err)
	}
	execURI := mustFileURI(t, execPath)

	s := NewLSPServer("8080", logrus.New())
	rootURI := mustFileURI(t, root)
	postLSPMethod(t, s, 1, "initialize", map[string]any{
		"processId": nil,
		"rootUri":   rootURI,
		"capabilities": map[string]any{},
	})

	resp := postLSPMethod(t, s, 2, "textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        execURI,
			"languageId": "forst",
			"version":    1,
			"text":       string(execSrc),
		},
	})
	assertNoGoImportDiagnostics(t, diagnosticsFromDidOpenResult(t, resp))
}
