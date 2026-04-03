package lsp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestHandleHealth_GET_ReturnsHealthyJSON(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("0", logrus.New())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	s.handleHealth(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "healthy" {
		t.Fatalf("status field = %v", body["status"])
	}
	if body["service"] != "forst-lsp" {
		t.Fatalf("service = %v", body["service"])
	}
	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("Content-Type = %q", ct)
	}
}

func TestHandleHealth_POST_MethodNotAllowed(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("0", logrus.New())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	s.handleHealth(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", rr.Code)
	}
}
