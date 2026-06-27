package devserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleVersion_marshalError_returns500(t *testing.T) {
	orig := jsonMarshalVersionPayload
	jsonMarshalVersionPayload = func(any) ([]byte, error) { return nil, fmt.Errorf("marshal") }
	t.Cleanup(func() { jsonMarshalVersionPayload = orig })

	s := testDevServer(t)
	rr := httptest.NewRecorder()
	s.handleVersion(rr, httptest.NewRequest(http.MethodGet, "/version", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleVersion_getAndWrongMethod(t *testing.T) {
	s := testDevServer(t)

	rr := httptest.NewRecorder()
	s.handleVersion(rr, httptest.NewRequest(http.MethodGet, "/version", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /version: %d %s", rr.Code, rr.Body.String())
	}
	var resp DevServerResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode version response: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %+v", resp)
	}
	var payload map[string]string
	if err := json.Unmarshal(resp.Result, &payload); err != nil {
		t.Fatalf("decode version payload: %v", err)
	}
	if payload["contractVersion"] != HTTPContractVersion {
		t.Fatalf("expected contractVersion %q, got %q", HTTPContractVersion, payload["contractVersion"])
	}

	rr2 := httptest.NewRecorder()
	s.handleVersion(rr2, httptest.NewRequest(http.MethodPost, "/version", nil))
	if rr2.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /version expected 405, got %d", rr2.Code)
	}
}
