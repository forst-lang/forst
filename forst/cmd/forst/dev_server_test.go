package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"forst/internal/codegen/layout"
	"forst/internal/compiler"

	"github.com/sirupsen/logrus"
)

func writeBlockingInvokeSocket(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, ".forst")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	sock := filepath.Join(dir, "invoke.sock")
	if err := os.WriteFile(sock, []byte("blocking"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func testDevServer(t *testing.T) *DevServer {
	t.Helper()
	log := logrus.New()
	log.SetOutput(io.Discard)
	cfg := DefaultConfig()
	cfg.Server.ReadTimeout = 1
	cfg.Server.WriteTimeout = 1
	falseVal := false
	cfg.Dev.WatchGenerate = &falseVal
	dir := t.TempDir()
	comp := compiler.New(cfg.ToCompilerArgs(), log)
	return NewHTTPServer("0", comp, log, cfg, dir)
}

// newInvokeTestRequest builds an httptest invoke request with loopback Host and JSON Content-Type.
func newInvokeTestRequest(method, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, target, body)
	req.Host = "127.0.0.1"
	req.RemoteAddr = "127.0.0.1:12345"
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func writeTestGeneratedTypes(t *testing.T, root, content string) {
	t.Helper()
	typesDir := filepath.Join(layout.NewRoot(root).ClientDir(), "dist")
	if err := os.MkdirAll(typesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(typesDir, "types.d.ts"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestNewHTTPServer_initializesTypesCache(t *testing.T) {
	s := testDevServer(t)
	if s.typesCache == nil {
		t.Fatal("typesCache must be non-nil for /types caching")
	}
}
