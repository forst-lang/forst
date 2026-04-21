package devserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"forst/internal/discovery"
	"forst/internal/executor"
)

type nonFlushingResponseWriter struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (w *nonFlushingResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *nonFlushingResponseWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(p)
}

func (w *nonFlushingResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}

type stubDevExecutor struct {
	executeFn          func(string, string, json.RawMessage) (*executor.ExecutionResult, error)
	executeStreamingFn func(context.Context, string, string, json.RawMessage) (<-chan executor.StreamingResult, error)
}

func (s *stubDevExecutor) ExecuteFunction(packageName, functionName string, args json.RawMessage) (*executor.ExecutionResult, error) {
	if s.executeFn != nil {
		return s.executeFn(packageName, functionName, args)
	}
	return nil, fmt.Errorf("stub: no executeFn")
}

func (s *stubDevExecutor) ExecuteStreamingFunction(ctx context.Context, packageName, functionName string, args json.RawMessage) (<-chan executor.StreamingResult, error) {
	if s.executeStreamingFn != nil {
		return s.executeStreamingFn(ctx, packageName, functionName, args)
	}
	return nil, fmt.Errorf("stub: no executeStreamingFn")
}

func TestHandleInvoke_wrongMethod(t *testing.T) {
	s := testDevServer(t)
	rr := httptest.NewRecorder()
	s.handleInvoke(rr, httptest.NewRequest(http.MethodGet, "/invoke", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET /invoke: %d", rr.Code)
	}
}

func TestHandleInvoke_invalidJSON(t *testing.T) {
	s := testDevServer(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/invoke", bytes.NewBufferString("not-json"))
	s.handleInvoke(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body %s", rr.Code, rr.Body.String())
	}
}

func TestHandleInvoke_packageNotFound(t *testing.T) {
	s := testDevServer(t)
	s.functions = make(map[string]map[string]discovery.FunctionInfo)

	body := bytes.NewBufferString(`{"package":"missing","function":"Fn","args":null}`)
	rr := httptest.NewRecorder()
	s.handleInvoke(rr, httptest.NewRequest(http.MethodPost, "/invoke", body))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleInvoke_functionNotFound(t *testing.T) {
	s := testDevServer(t)
	s.functions = map[string]map[string]discovery.FunctionInfo{
		"mypkg": {},
	}

	body := bytes.NewBufferString(`{"package":"mypkg","function":"Nope","args":null}`)
	rr := httptest.NewRecorder()
	s.handleInvoke(rr, httptest.NewRequest(http.MethodPost, "/invoke", body))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleInvoke_streamingNotSupported_returns400(t *testing.T) {
	t.Parallel()
	s := testDevServer(t)
	s.functions = map[string]map[string]discovery.FunctionInfo{
		"mypkg": {
			"NoStream": {
				Package:           "mypkg",
				Name:              "NoStream",
				SupportsStreaming: false,
			},
		},
	}
	body := `{"package":"mypkg","function":"NoStream","args":[],"streaming":true}`
	rr := httptest.NewRecorder()
	s.handleInvoke(rr, httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(body)))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp DevServerResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Success || !strings.Contains(resp.Error, "does not support streaming") {
		t.Fatalf("expected streaming error envelope, got %+v", resp)
	}
}

func TestHandleInvoke_streamingRequiresFlusher_returns500(t *testing.T) {
	s := testDevServer(t)
	s.functions = map[string]map[string]discovery.FunctionInfo{
		"mypkg": {
			"StreamFn": {
				Package:           "mypkg",
				Name:              "StreamFn",
				SupportsStreaming: true,
			},
		},
	}
	reqBody := `{"package":"mypkg","function":"StreamFn","args":[],"streaming":true}`
	writer := &nonFlushingResponseWriter{}

	s.handleInvoke(writer, httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(reqBody)))
	if writer.status != http.StatusInternalServerError {
		t.Fatalf("expected 500 when flusher missing, got %d body=%s", writer.status, writer.body.String())
	}
	if !strings.Contains(writer.body.String(), "Streaming not supported by server") {
		t.Fatalf("unexpected response body: %s", writer.body.String())
	}
}

func TestHandleInvoke_executeFunctionFailure_returns500(t *testing.T) {
	s := testDevServer(t)
	s.functions = map[string]map[string]discovery.FunctionInfo{
		"mypkg": {
			"Fn": {
				Package: "mypkg",
				Name:    "Fn",
			},
		},
	}
	body := `{"package":"mypkg","function":"Fn","args":[]}`
	rr := httptest.NewRecorder()
	s.handleInvoke(rr, httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(body)))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when execution fails, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Function execution failed") {
		t.Fatalf("unexpected error response: %s", rr.Body.String())
	}
}

func TestHandleInvoke_streamingExecuteError_returns500(t *testing.T) {
	s := testDevServer(t)
	s.functions = map[string]map[string]discovery.FunctionInfo{
		"mypkg": {
			"StreamFn": {Package: "mypkg", Name: "StreamFn", SupportsStreaming: true},
		},
	}
	s.fnExec = &stubDevExecutor{
		executeStreamingFn: func(context.Context, string, string, json.RawMessage) (<-chan executor.StreamingResult, error) {
			return nil, fmt.Errorf("stream boom")
		},
	}
	rr := httptest.NewRecorder()
	body := `{"package":"mypkg","function":"StreamFn","args":[],"streaming":true}`
	s.handleInvoke(rr, httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(body)))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleInvoke_executeFunctionSuccess_returns200(t *testing.T) {
	t.Parallel()
	s := testDevServer(t)
	s.functions = map[string]map[string]discovery.FunctionInfo{
		"mypkg": {
			"Fn": {Package: "mypkg", Name: "Fn"},
		},
	}
	s.fnExec = &stubDevExecutor{
		executeFn: func(_, _ string, _ json.RawMessage) (*executor.ExecutionResult, error) {
			return &executor.ExecutionResult{
				Success: true,
				Result:  json.RawMessage(`{"ok":true}`),
			}, nil
		},
	}
	body := `{"package":"mypkg","function":"Fn","args":[]}`
	rr := httptest.NewRecorder()
	s.handleInvoke(rr, httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(body)))
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var resp DevServerResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success || string(resp.Result) != `{"ok":true}` {
		t.Fatalf("unexpected body: %+v", resp)
	}
}

func TestHandleInvoke_streamingSuccess_writesNDJSON(t *testing.T) {
	t.Parallel()
	s := testDevServer(t)
	s.functions = map[string]map[string]discovery.FunctionInfo{
		"mypkg": {
			"StreamFn": {
				Package: "mypkg", Name: "StreamFn", SupportsStreaming: true,
			},
		},
	}
	ch := make(chan executor.StreamingResult, 2)
	ch <- executor.StreamingResult{Data: map[string]int{"n": 1}, Status: "ok"}
	ch <- executor.StreamingResult{Data: map[string]int{"n": 2}, Status: "ok"}
	close(ch)
	s.fnExec = &stubDevExecutor{
		executeStreamingFn: func(context.Context, string, string, json.RawMessage) (<-chan executor.StreamingResult, error) {
			return ch, nil
		},
	}
	reqBody := `{"package":"mypkg","function":"StreamFn","args":[],"streaming":true}`
	rr := httptest.NewRecorder()
	s.handleInvoke(rr, httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(reqBody)))
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

// errRecorder wraps httptest.ResponseRecorder and fails writes after enough bytes so a second JSON encode fails.
type errAfterManyBytes struct {
	*httptest.ResponseRecorder
	limit int
	n     int
}

func (w *errAfterManyBytes) Write(p []byte) (int, error) {
	if w.n > w.limit {
		return 0, fmt.Errorf("simulated write failure")
	}
	n, err := w.ResponseRecorder.Write(p)
	w.n += n
	if w.n > w.limit {
		return n, fmt.Errorf("simulated write failure")
	}
	return n, err
}

func (w *errAfterManyBytes) Flush() {
	w.ResponseRecorder.Flush()
}

func TestHandleInvoke_streamingEncodeError_stopsAfterFirstChunk(t *testing.T) {
	s := testDevServer(t)
	s.functions = map[string]map[string]discovery.FunctionInfo{
		"mypkg": {
			"StreamFn": {
				Package: "mypkg", Name: "StreamFn", SupportsStreaming: true,
			},
		},
	}
	ch := make(chan executor.StreamingResult, 3)
	ch <- executor.StreamingResult{Data: "first", Status: "ok"}
	ch <- executor.StreamingResult{Data: "second", Status: "ok"}
	close(ch)
	s.fnExec = &stubDevExecutor{
		executeStreamingFn: func(context.Context, string, string, json.RawMessage) (<-chan executor.StreamingResult, error) {
			return ch, nil
		},
	}
	base := httptest.NewRecorder()
	rr := &errAfterManyBytes{ResponseRecorder: base, limit: 40}
	reqBody := `{"package":"mypkg","function":"StreamFn","args":[],"streaming":true}`
	s.handleInvoke(rr, httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(reqBody)))
	if base.Code != 0 && base.Code != http.StatusOK {
		t.Fatalf("unexpected code %d", base.Code)
	}
}
