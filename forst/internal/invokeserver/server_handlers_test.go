package invokeserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"forst/internal/discovery"
	"forst/internal/invokedispatch"
)

func TestHandleHealth_wrongMethod(t *testing.T) {
	s := newTestServer(t, &stubBackend{})
	rr := httptest.NewRecorder()
	s.HandleHealth(rr, httptest.NewRequest(http.MethodPost, "/health", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /health: %d", rr.Code)
	}
}

func TestHandleVersion_getAndWrongMethod(t *testing.T) {
	s := newTestServer(t, &stubBackend{})

	rr := httptest.NewRecorder()
	s.HandleVersion(rr, httptest.NewRequest(http.MethodGet, "/version", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /version: %d %s", rr.Code, rr.Body.String())
	}
	var resp Response
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil || !resp.Success {
		t.Fatalf("body: %+v err=%v", resp, err)
	}

	rr2 := httptest.NewRecorder()
	s.HandleVersion(rr2, httptest.NewRequest(http.MethodPost, "/version", nil))
	if rr2.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /version: %d", rr2.Code)
	}
}

func TestHandleVersion_marshalError(t *testing.T) {
	orig := marshalVersionPayload
	marshalVersionPayload = func(VersionInfo) ([]byte, error) { return nil, fmt.Errorf("marshal") }
	t.Cleanup(func() { marshalVersionPayload = orig })

	s := newTestServer(t, &stubBackend{})
	rr := httptest.NewRecorder()
	s.HandleVersion(rr, httptest.NewRequest(http.MethodGet, "/version", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleFunctions_get_success(t *testing.T) {
	t.Parallel()
	backend := &stubBackend{
		functions: map[string]map[string]discovery.FunctionInfo{
			"main": {"Echo": {Package: "main", Name: "Echo", Runnable: true}},
		},
	}
	s := newTestServer(t, backend)
	rr := httptest.NewRecorder()
	s.HandleFunctions(rr, httptest.NewRequest(http.MethodGet, "/functions", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /functions: %d %s", rr.Code, rr.Body.String())
	}
	var resp Response
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil || !resp.Success {
		t.Fatalf("resp: %+v err=%v", resp, err)
	}
}

func TestHandleFunctions_wrongMethod(t *testing.T) {
	s := newTestServer(t, &stubBackend{})
	rr := httptest.NewRecorder()
	s.HandleFunctions(rr, httptest.NewRequest(http.MethodPost, "/functions", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /functions: %d", rr.Code)
	}
}

func TestHandleFunctions_refreshFailure(t *testing.T) {
	backend := &stubBackend{refreshErr: fmt.Errorf("discover boom")}
	s := newTestServer(t, backend)
	rr := httptest.NewRecorder()
	s.HandleFunctions(rr, httptest.NewRequest(http.MethodGet, "/functions", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleFunctions_marshalError(t *testing.T) {
	orig := marshalFunctionList
	marshalFunctionList = func([]discovery.FunctionInfo) ([]byte, error) { return nil, fmt.Errorf("no") }
	t.Cleanup(func() { marshalFunctionList = orig })

	backend := &stubBackend{
		functions: map[string]map[string]discovery.FunctionInfo{
			"p": {"F": {Package: "p", Name: "F"}},
		},
	}
	s := newTestServer(t, backend)
	rr := httptest.NewRecorder()
	s.HandleFunctions(rr, httptest.NewRequest(http.MethodGet, "/functions", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleInvoke_wrongMethod(t *testing.T) {
	s := newTestServer(t, &stubBackend{})
	rr := httptest.NewRecorder()
	s.HandleInvoke(rr, httptest.NewRequest(http.MethodGet, "/invoke", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET /invoke: %d", rr.Code)
	}
}

func TestHandleInvoke_invalidJSON(t *testing.T) {
	s := newTestServer(t, &stubBackend{})
	rr := httptest.NewRecorder()
	s.HandleInvoke(rr, httptest.NewRequest(http.MethodPost, "/invoke", bytes.NewBufferString("not-json")))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body %s", rr.Code, rr.Body.String())
	}
}

func TestHandleInvoke_readBodyError(t *testing.T) {
	s := newTestServer(t, &stubBackend{})
	rr := httptest.NewRecorder()
	s.HandleInvoke(rr, httptest.NewRequest(http.MethodPost, "/invoke", io.NopCloser(failReader{})))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body %s", rr.Code, rr.Body.String())
	}
}

func TestHandleInvoke_oversizedBody(t *testing.T) {
	s := newTestServer(t, &stubBackend{})
	s.SetMaxRequestSize(64)
	body := strings.Repeat("x", 128)
	rr := httptest.NewRecorder()
	s.HandleInvoke(rr, httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(body)))
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("want 413, got %d body %s", rr.Code, rr.Body.String())
	}
}

func TestHandleInvoke_packageNotFound(t *testing.T) {
	s := newTestServer(t, &stubBackend{functions: map[string]map[string]discovery.FunctionInfo{}})
	body := bytes.NewBufferString(`{"package":"missing","function":"Fn","args":null}`)
	rr := httptest.NewRecorder()
	s.HandleInvoke(rr, httptest.NewRequest(http.MethodPost, "/invoke", body))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleInvoke_functionNotFound(t *testing.T) {
	s := newTestServer(t, &stubBackend{
		functions: map[string]map[string]discovery.FunctionInfo{"mypkg": {}},
	})
	body := bytes.NewBufferString(`{"package":"mypkg","function":"Nope","args":null}`)
	rr := httptest.NewRecorder()
	s.HandleInvoke(rr, httptest.NewRequest(http.MethodPost, "/invoke", body))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleInvoke_executeFailure(t *testing.T) {
	backend := &stubBackend{
		functions: map[string]map[string]discovery.FunctionInfo{
			"mypkg": {"Fn": {Package: "mypkg", Name: "Fn"}},
		},
		invoke: func(_, _ string, _ json.RawMessage) (*invokedispatch.InvokeResult, error) {
			return nil, fmt.Errorf("boom")
		},
	}
	s := newTestServer(t, backend)
	rr := httptest.NewRecorder()
	s.HandleInvoke(rr, httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(`{"package":"mypkg","function":"Fn","args":[]}`)))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleInvoke_streamingNotSupported(t *testing.T) {
	t.Parallel()
	backend := &stubBackend{
		functions: map[string]map[string]discovery.FunctionInfo{
			"mypkg": {"NoStream": {Package: "mypkg", Name: "NoStream", SupportsStreaming: false}},
		},
	}
	s := newTestServer(t, backend)
	rr := httptest.NewRecorder()
	body := `{"package":"mypkg","function":"NoStream","args":[],"streaming":true}`
	s.HandleInvoke(rr, httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(body)))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleInvoke_streamingRequiresFlusher(t *testing.T) {
	backend := &stubBackend{
		functions: map[string]map[string]discovery.FunctionInfo{
			"mypkg": {"StreamFn": {Package: "mypkg", Name: "StreamFn", SupportsStreaming: true}},
		},
	}
	s := newTestServer(t, backend)
	writer := &nonFlushingResponseWriter{}
	s.HandleInvoke(writer, httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(`{"package":"mypkg","function":"StreamFn","args":[],"streaming":true}`)))
	if writer.status != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", writer.status, writer.body.String())
	}
}

func TestHandleInvoke_streamingExecuteError(t *testing.T) {
	backend := &stubBackend{
		functions: map[string]map[string]discovery.FunctionInfo{
			"mypkg": {"StreamFn": {Package: "mypkg", Name: "StreamFn", SupportsStreaming: true}},
		},
		invokeStream: func(context.Context, string, string, json.RawMessage) (<-chan invokedispatch.StreamChunk, error) {
			return nil, fmt.Errorf("stream boom")
		},
	}
	s := newTestServer(t, backend)
	rr := httptest.NewRecorder()
	s.HandleInvoke(rr, httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(`{"package":"mypkg","function":"StreamFn","args":[],"streaming":true}`)))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleInvoke_streamingSuccess_writesNDJSON(t *testing.T) {
	t.Parallel()
	ch := make(chan invokedispatch.StreamChunk, 2)
	ch <- invokedispatch.StreamChunk{Data: map[string]int{"n": 1}, Status: "ok"}
	ch <- invokedispatch.StreamChunk{Data: map[string]int{"n": 2}, Status: "ok"}
	close(ch)
	backend := &stubBackend{
		functions: map[string]map[string]discovery.FunctionInfo{
			"mypkg": {"StreamFn": {Package: "mypkg", Name: "StreamFn", SupportsStreaming: true}},
		},
		invokeStream: func(context.Context, string, string, json.RawMessage) (<-chan invokedispatch.StreamChunk, error) {
			return ch, nil
		},
	}
	s := newTestServer(t, backend)
	rr := httptest.NewRecorder()
	s.HandleInvoke(rr, httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(`{"package":"mypkg","function":"StreamFn","args":[],"streaming":true}`)))
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Fatalf("Content-Type: %q", ct)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `"n":1`) || !strings.Contains(body, `"n":2`) {
		t.Fatalf("unexpected streaming body: %s", body)
	}
}

func TestHandleInvoke_streamingEncodeError(t *testing.T) {
	ch := make(chan invokedispatch.StreamChunk, 3)
	ch <- invokedispatch.StreamChunk{Data: "first", Status: "ok"}
	ch <- invokedispatch.StreamChunk{Data: "second", Status: "ok"}
	close(ch)
	log := &stubLogger{}
	backend := &stubBackend{
		functions: map[string]map[string]discovery.FunctionInfo{
			"mypkg": {"StreamFn": {Package: "mypkg", Name: "StreamFn", SupportsStreaming: true}},
		},
		invokeStream: func(context.Context, string, string, json.RawMessage) (<-chan invokedispatch.StreamChunk, error) {
			return ch, nil
		},
	}
	cfg := Config{Host: "127.0.0.1", Port: "0", Runtime: "embedded"}
	s := New(cfg, backend, DefaultEmbeddedVersion(), log)
	base := httptest.NewRecorder()
	rr := &errAfterManyBytes{ResponseRecorder: base, limit: 40}
	s.HandleInvoke(rr, httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(`{"package":"mypkg","function":"StreamFn","args":[],"streaming":true}`)))
	if len(log.errors) == 0 {
		t.Fatal("expected encode error log")
	}
}

func TestSendJSON_CORS(t *testing.T) {
	backend := &stubBackend{}
	s := newTestServer(t, backend, func(c *Config) { c.CORS = true })
	rr := httptest.NewRecorder()
	s.HandleHealth(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("CORS header missing: %+v", rr.Header())
	}
}

func TestNew_defaultContractVersion(t *testing.T) {
	s := New(Config{}, &stubBackend{}, VersionInfo{}, nil)
	if s.version.ContractVersion != HTTPContractVersion {
		t.Fatalf("contract = %q", s.version.ContractVersion)
	}
}
