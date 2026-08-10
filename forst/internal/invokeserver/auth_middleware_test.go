package invokeserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/discovery"
)

func TestAuthMiddleware_missingProofRejected401(t *testing.T) {
	s := New(Config{Host: "127.0.0.1", Port: "0", Runtime: "embedded"}, &stubBackend{
		functions: map[string]map[string]discovery.FunctionInfo{
			"main": {"Fn": {}},
		},
	}, DefaultEmbeddedVersion(), nil)

	mux := http.NewServeMux()
	s.RegisterRoutes(mux)
	handler := s.authMiddleware(mux)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, newInvokeHTTPRequest(http.MethodPost, "/invoke", strings.NewReader(`{"package":"main","function":"Fn","args":[]}`)))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAuthMiddleware_validProofSucceeds(t *testing.T) {
	s := New(Config{Host: "127.0.0.1", Port: "0", Runtime: "embedded"}, &stubBackend{
		functions: map[string]map[string]discovery.FunctionInfo{
			"main": {"Fn": {}},
		},
	}, DefaultEmbeddedVersion(), nil)

	mux := http.NewServeMux()
	s.RegisterRoutes(mux)
	handler := s.authMiddleware(mux)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, authenticatedRequest(t, s, http.MethodPost, "/invoke", strings.NewReader(`{"package":"main","function":"Fn","args":[]}`)))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestServer_handleChallenge_returnsNonceAndExpiry(t *testing.T) {
	s := New(Config{Host: "127.0.0.1", Port: "0", Runtime: "embedded"}, &stubBackend{}, DefaultEmbeddedVersion(), nil)
	rr := httptest.NewRecorder()
	s.HandleChallenge(rr, httptest.NewRequest(http.MethodGet, "/invoke/challenge", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestWriteInvokeReady_fileMode0600(t *testing.T) {
	dir := t.TempDir()
	if err := writeInvokeReady(dir, Config{Host: "127.0.0.1", Port: "8081", Runtime: "embedded"}, 1); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, ".forst", "invoke.ready"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
}
