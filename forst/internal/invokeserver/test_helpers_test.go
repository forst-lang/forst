package invokeserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func newInvokeHTTPRequest(method, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, target, body)
	req.Host = "127.0.0.1"
	req.RemoteAddr = "127.0.0.1:12345"
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func authenticatedRequest(t *testing.T, s *Server, method, target string, body io.Reader) *http.Request {
	t.Helper()
	req := newInvokeHTTPRequest(method, target, body)
	if !s.authEnabled() || s.nonces == nil || s.auth == nil {
		return req
	}
	token, generation := s.CurrentAuth()
	nonce, _, err := s.nonces.issue(time.Now())
	if err != nil {
		t.Fatalf("issue nonce: %v", err)
	}
	req.Header.Set(HeaderInvokeNonce, nonce)
	req.Header.Set(HeaderInvokeGeneration, strconv.FormatUint(generation, 10))
	req.Header.Set(HeaderInvokeProof, encodeInvokeProof(computeInvokeProof(token, generation, nonce)))
	return req
}
