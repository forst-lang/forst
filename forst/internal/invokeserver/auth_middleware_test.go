package invokeserver

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

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

// writeHeaderCounter records how many times WriteHeader is called so auth failures
// cannot regress into a superfluous second status write via sendJSON.
type writeHeaderCounter struct {
	http.ResponseWriter
	headerWrites int
	status       int
}

func (w *writeHeaderCounter) WriteHeader(statusCode int) {
	w.headerWrites++
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func TestAuthMiddleware_missingProofSingleWriteHeader(t *testing.T) {
	s := New(Config{Host: "127.0.0.1", Port: "0", Runtime: "embedded"}, &stubBackend{
		functions: map[string]map[string]discovery.FunctionInfo{
			"main": {"Fn": {}},
		},
	}, DefaultEmbeddedVersion(), nil)

	mux := http.NewServeMux()
	s.RegisterRoutes(mux)
	handler := s.authMiddleware(mux)

	rr := httptest.NewRecorder()
	counter := &writeHeaderCounter{ResponseWriter: rr}
	handler.ServeHTTP(counter, newInvokeHTTPRequest(http.MethodPost, "/invoke", strings.NewReader(`{"package":"main","function":"Fn","args":[]}`)))
	if counter.headerWrites != 1 {
		t.Fatalf("WriteHeader calls = %d, want 1", counter.headerWrites)
	}
	if counter.status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", counter.status, http.StatusUnauthorized)
	}
	var envelope Response
	if err := json.Unmarshal(rr.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode body: %v body=%s", err, rr.Body.String())
	}
	if envelope.Success {
		t.Fatal("expected Success=false")
	}
	if envelope.Error != "unauthorized" {
		t.Fatalf("error = %q, want unauthorized", envelope.Error)
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
	var envelope Response
	if err := json.Unmarshal(rr.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	var challenge ChallengeResponse
	if err := json.Unmarshal(envelope.Result, &challenge); err != nil {
		t.Fatal(err)
	}
	if challenge.Nonce == "" {
		t.Fatal("expected non-empty nonce")
	}
	expiresAt, err := time.Parse(time.RFC3339, challenge.ExpiresAt)
	if err != nil {
		t.Fatalf("expiresAt parse: %v", err)
	}
	if !expiresAt.After(time.Now().UTC().Add(-time.Second)) {
		t.Fatalf("expiresAt = %v", expiresAt)
	}
}

func TestWriteInvokeReady_fileMode0600(t *testing.T) {
	dir := t.TempDir()
	if err := writeInvokeReady(dir, Config{Host: "127.0.0.1", Port: "8081", Runtime: "embedded"}, 1, ""); err != nil {
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

func TestAuthMiddleware_unixPeerCredAllowsAuthenticatedInvoke(t *testing.T) {
	if !PeerCredEnforced() {
		t.Skip("peercred not enforced on this platform")
	}

	workDir := t.TempDir()
	cfg := Config{
		Runtime:      "embedded",
		BoundaryRoot: workDir,
		Transport:    transportUnix,
	}
	ApplyListenDefaults(&cfg, workDir)

	s := New(cfg, &stubBackend{
		functions: map[string]map[string]discovery.FunctionInfo{
			"main": {"Fn": {}},
		},
	}, DefaultEmbeddedVersion(), nil)
	if err := s.StartAsync(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Stop() })

	socketPath := s.BoundAddr()
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
			},
		},
	}

	challengeResp, err := client.Get("http://unix/invoke/challenge")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = challengeResp.Body.Close() }()
	if challengeResp.StatusCode != http.StatusOK {
		t.Fatalf("challenge status = %d", challengeResp.StatusCode)
	}
	var envelope Response
	if err := json.NewDecoder(challengeResp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	var challenge ChallengeResponse
	if err := json.Unmarshal(envelope.Result, &challenge); err != nil {
		t.Fatal(err)
	}
	if challenge.Nonce == "" {
		t.Fatal("expected challenge nonce")
	}

	token, generation := s.CurrentAuth()
	req, err := http.NewRequest(http.MethodPost, "http://unix/invoke", strings.NewReader(`{"package":"main","function":"Fn","args":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(HeaderInvokeNonce, challenge.Nonce)
	req.Header.Set(HeaderInvokeGeneration, strconv.FormatUint(generation, 10))
	req.Header.Set(HeaderInvokeProof, ComputeInvokeProofForTest(token, generation, challenge.Nonce))

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body=%s", resp.StatusCode, body)
	}
}
