package lsp

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"forst/internal/httpbody"

	"github.com/sirupsen/logrus"
)

func TestHandleLSP_rejectsOversizedBody(t *testing.T) {
	s := NewLSPServer("8080", logrus.New())
	s.maxRequestBodyBytes = 32

	body := strings.Repeat("x", 64)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	rr := httptest.NewRecorder()

	s.handleLSP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d want 413 body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleLSP_acceptsBodyUnderLimit(t *testing.T) {
	s := NewLSPServer("8080", logrus.New())
	s.maxRequestBodyBytes = 1024

	payload := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
	rr := httptest.NewRecorder()

	s.handleLSP(rr, req)

	if rr.Code == http.StatusRequestEntityTooLarge {
		t.Fatalf("unexpected 413 for small body")
	}
}

func TestReadAll_usesDefaultWhenLSPServerMaxZero(t *testing.T) {
	_, err := httpbody.ReadAll(bytes.NewReader([]byte("ok")), 0)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
}

func TestLimitReader_interface(t *testing.T) {
	r := httpbody.LimitReader(strings.NewReader("ab"), 1)
	_, err := io.ReadAll(r)
	if err == nil || !httpbody.IsTooLarge(err) {
		t.Fatalf("expected too large, got %v", err)
	}
}
