package devserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendJSONResponse_setsCORSWhenEnabled(t *testing.T) {
	s := testDevServer(t)
	s.httpOpts.CORS = true
	rr := httptest.NewRecorder()
	s.sendJSONResponse(rr, DevServerResponse{Success: true})
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected CORS header, got %v", rr.Header())
	}
}

func TestSendJSONResponse_doesNotSetCORSWhenDisabled(t *testing.T) {
	s := testDevServer(t)
	s.httpOpts.CORS = false
	rr := httptest.NewRecorder()
	s.sendJSONResponse(rr, DevServerResponse{Success: true})
	if rr.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no CORS header, got %v", rr.Header())
	}
}

type jsonErrorResponseWriter struct {
	header http.Header
	code   int
}

func (w *jsonErrorResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *jsonErrorResponseWriter) Write([]byte) (int, error) {
	return 0, fmt.Errorf("write blocked")
}

func (w *jsonErrorResponseWriter) WriteHeader(statusCode int) {
	w.code = statusCode
}

func TestSendJSONResponse_encoderError_fallsBackToHTTPError(t *testing.T) {
	s := testDevServer(t)
	w := &jsonErrorResponseWriter{}
	s.sendJSONResponse(w, DevServerResponse{Success: true})
	if w.code != http.StatusInternalServerError {
		t.Fatalf("expected 500 from http.Error, got %d", w.code)
	}
}

func TestSendError_setsStatusAndJSONEnvelope(t *testing.T) {
	s := testDevServer(t)
	rr := httptest.NewRecorder()
	s.sendError(rr, "upstream gateway timed out", http.StatusBadGateway)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status: got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type: want application/json, got %q", ct)
	}
	var resp DevServerResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Success || resp.Error != "upstream gateway timed out" {
		t.Fatalf("unexpected error envelope: %+v", resp)
	}
}
