package devserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleHealth_getOk_postNotAllowed(t *testing.T) {
	s := testDevServer(t)

	rr := httptest.NewRecorder()
	s.handleHealth(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /health: %d", rr.Code)
	}
	var resp DevServerResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil || !resp.Success {
		t.Fatalf("body: %+v err %v", resp, err)
	}

	rr2 := httptest.NewRecorder()
	s.handleHealth(rr2, httptest.NewRequest(http.MethodPost, "/health", nil))
	if rr2.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /health: %d", rr2.Code)
	}
}
