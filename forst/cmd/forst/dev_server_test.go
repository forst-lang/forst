package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"forst/internal/compiler"
	"forst/internal/discovery"

	"github.com/sirupsen/logrus"
)

func testDevServer(t *testing.T) *DevServer {
	t.Helper()
	log := logrus.New()
	log.SetOutput(io.Discard)
	cfg := DefaultConfig()
	cfg.Server.ReadTimeout = 1
	cfg.Server.WriteTimeout = 1
	comp := compiler.New(cfg.ToCompilerArgs(), log)
	return NewHTTPServer("0", comp, log, cfg, t.TempDir())
}

func TestNewHTTPServer_initializesTypesCache(t *testing.T) {
	s := testDevServer(t)
	if s.typesCache == nil {
		t.Fatal("typesCache must be non-nil for /types caching")
	}
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

func TestNewTypeScriptGenerator_GenerateTypesForFunctions_emptyDiscoveryReturnsHeaderOnly(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	tg := NewTypeScriptGenerator(log)
	out, err := tg.GenerateTypesForFunctions(nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Auto-generated types for Forst client") {
		t.Fatalf("expected header in types output, got %q", out)
	}
}

func TestTypeScriptGenerator_generateTypesForFile_readsParsesAndTransforms(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "echo.ft")
	if err := os.WriteFile(ft, []byte(generateTestMinimalValidForst), 0644); err != nil {
		t.Fatal(err)
	}

	log := logrus.New()
	log.SetOutput(io.Discard)
	tg := NewTypeScriptGenerator(log)
	types, funcs, pkg, err := tg.generateTypesForFile(ft)
	if err != nil {
		t.Fatal(err)
	}
	if pkg != "main" {
		t.Fatalf("package: %q", pkg)
	}
	if len(funcs) == 0 || len(types) == 0 {
		t.Fatalf("expected types and functions, got types=%d funcs=%d", len(types), len(funcs))
	}
}

func TestTypeScriptGenerator_generateTypesForFile_missingFile(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	tg := NewTypeScriptGenerator(log)
	_, _, _, err := tg.generateTypesForFile(filepath.Join(t.TempDir(), "nope.ft"))
	if err == nil || !strings.Contains(err.Error(), "failed to read") {
		t.Fatalf("expected read error, got %v", err)
	}
}

func TestDevServer_Stop_nilServerNoop(t *testing.T) {
	s := &DevServer{}
	if err := s.Stop(); err != nil {
		t.Fatal(err)
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
