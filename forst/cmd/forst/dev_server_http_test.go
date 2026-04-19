package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestHandleFunctions_getOk_wrongMethod(t *testing.T) {
	s := testDevServer(t)
	rr := httptest.NewRecorder()
	s.handleFunctions(rr, httptest.NewRequest(http.MethodPost, "/functions", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /functions: %d", rr.Code)
	}
}

func TestHandleFunctions_get_success_returnsJSONResultWithFunctionList(t *testing.T) {
	t.Parallel()
	s := testDevServer(t)
	rr := httptest.NewRecorder()
	s.handleFunctions(rr, httptest.NewRequest(http.MethodGet, "/functions", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /functions: %d %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type: want application/json, got %q", ct)
	}
	var resp DevServerResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %+v", resp)
	}
	var list []discovery.FunctionInfo
	if err := json.Unmarshal(resp.Result, &list); err != nil {
		t.Fatalf("result JSON: %q err %v", resp.Result, err)
	}
	if list == nil {
		t.Fatal("expected result to decode to JSON array (empty is ok), got nil slice")
	}
}

func TestHandleFunctions_discoveryFailure_returns500(t *testing.T) {
	s := testDevServer(t)
	s.discoverer = discovery.NewDiscoverer(t.TempDir(), s.log, nil)
	rr := httptest.NewRecorder()

	s.handleFunctions(rr, httptest.NewRequest(http.MethodGet, "/functions", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for discovery failure, got %d body=%s", rr.Code, rr.Body.String())
	}
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
	if payload["contractVersion"] != devHTTPContractVersion {
		t.Fatalf("expected contractVersion %q, got %q", devHTTPContractVersion, payload["contractVersion"])
	}

	rr2 := httptest.NewRecorder()
	s.handleVersion(rr2, httptest.NewRequest(http.MethodPost, "/version", nil))
	if rr2.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /version expected 405, got %d", rr2.Code)
	}
}

func TestHandleTypes_get_returnsJSON_wrongMethod(t *testing.T) {
	s := testDevServer(t)
	s.typesGenerator = NewTypeScriptGenerator(s.log)

	rr := httptest.NewRecorder()
	s.handleTypes(rr, httptest.NewRequest(http.MethodGet, "/types", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /types: %d %s", rr.Code, rr.Body.String())
	}
	var resp DevServerResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil || !resp.Success {
		t.Fatalf("decode: %v resp=%+v", err, resp)
	}

	rr2 := httptest.NewRecorder()
	s.handleTypes(rr2, httptest.NewRequest(http.MethodPost, "/types", nil))
	if rr2.Code != http.StatusMethodNotAllowed {
		t.Fatalf("bad method: %d", rr2.Code)
	}
}

func TestHandleTypes_freshCache_returnsCachedWithoutGenerator(t *testing.T) {
	s := testDevServer(t)
	s.typesCacheMu.Lock()
	s.typesCache["types"] = "cached-types-content"
	s.lastTypesGen = time.Now()
	s.typesCacheMu.Unlock()

	rr := httptest.NewRecorder()
	s.handleTypes(rr, httptest.NewRequest(http.MethodGet, "/types", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /types: %d %s", rr.Code, rr.Body.String())
	}

	var resp DevServerResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %+v", resp)
	}
	if resp.Output != "cached-types-content" {
		t.Fatalf("expected cached output, got %q", resp.Output)
	}
}

func TestHandleTypes_forceRegenerate_overwritesCache(t *testing.T) {
	s := testDevServer(t)
	s.typesGenerator = NewTypeScriptGenerator(s.log)
	s.typesCacheMu.Lock()
	s.typesCache["types"] = "stale-cache"
	s.lastTypesGen = time.Now()
	s.typesCacheMu.Unlock()

	rr := httptest.NewRecorder()
	s.handleTypes(rr, httptest.NewRequest(http.MethodGet, "/types?force=true", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /types?force=true: %d %s", rr.Code, rr.Body.String())
	}

	var resp DevServerResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %+v", resp)
	}
	if resp.Output == "stale-cache" {
		t.Fatalf("expected regenerated output, got stale cache")
	}
}

func TestHandleTypes_staleCache_regeneratesWithoutForce(t *testing.T) {
	s := testDevServer(t)
	s.typesGenerator = NewTypeScriptGenerator(s.log)
	s.typesCacheMu.Lock()
	s.typesCache["types"] = "stale-cache"
	s.lastTypesGen = time.Now().Add(-6 * time.Minute)
	s.typesCacheMu.Unlock()

	rr := httptest.NewRecorder()
	s.handleTypes(rr, httptest.NewRequest(http.MethodGet, "/types", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /types stale cache: %d %s", rr.Code, rr.Body.String())
	}

	var resp DevServerResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %+v", resp)
	}
	if resp.Output == "stale-cache" {
		t.Fatalf("expected stale cache invalidation and regeneration")
	}
}

func TestSendJSONResponse_setsCORSWhenEnabled(t *testing.T) {
	s := testDevServer(t)
	s.config.Server.CORS = true
	rr := httptest.NewRecorder()
	s.sendJSONResponse(rr, DevServerResponse{Success: true})
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected CORS header, got %v", rr.Header())
	}
}

func TestSendJSONResponse_doesNotSetCORSWhenDisabled(t *testing.T) {
	s := testDevServer(t)
	s.config.Server.CORS = false
	rr := httptest.NewRecorder()
	s.sendJSONResponse(rr, DevServerResponse{Success: true})
	if rr.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no CORS header, got %v", rr.Header())
	}
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

func TestHandleTypes_discoveryFailureOnRegenerate_returns500(t *testing.T) {
	s := testDevServer(t)
	s.typesGenerator = NewTypeScriptGenerator(s.log)
	s.discoverer = discovery.NewDiscoverer(t.TempDir(), s.log, nil)

	rr := httptest.NewRecorder()
	s.handleTypes(rr, httptest.NewRequest(http.MethodGet, "/types?force=true", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d body=%s", rr.Code, rr.Body.String())
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
	// http.Error sets text/plain body; code should be 500.
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
