package devserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/compiler"
	"forst/internal/discovery"

	"github.com/sirupsen/logrus"
)

// Minimal valid Forst sources (same fixtures as cmd/forst generate tests).
const testMinimalValidForst = `package main

type EchoRequest = {
	message: String
}

func Echo(input EchoRequest) {
	return {
		echo: input.message,
		timestamp: 1234567890,
	}
}
`

const testSecondForstFile = `package main

type Ping = {
	ok: Bool
}

func PingServer(input Ping) {
	return { pong: input.ok }
}
`

// walkForstConfig finds all .ft files under root (minimal subset of main ForstConfig behavior).
type walkForstConfig struct{}

func (walkForstConfig) FindForstFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(path), ".ft") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
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

func TestHandleFunctions_twoPublicFuncsSamePackage(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.ft"), []byte(testMinimalValidForst), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.ft"), []byte(testSecondForstFile), 0o644); err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetOutput(io.Discard)
	args := compiler.Args{
		LogLevel:           "info",
		ReportPhases:       false,
		ReportMemoryUsage:  false,
		ExportStructFields: false,
	}
	comp := compiler.New(args, log)
	cfg := walkForstConfig{}
	s := NewHTTPServer("0", comp, log, cfg, BuildInfo{Version: "dev", Commit: "unknown", Date: "unknown"}, HTTPOpts{ReadTimeoutSec: 1, WriteTimeoutSec: 1, CORS: true}, root)

	rr := httptest.NewRecorder()
	s.handleFunctions(rr, httptest.NewRequest(http.MethodGet, "/functions", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /functions: %d %s", rr.Code, rr.Body.String())
	}
	var resp DevServerResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	var list []discovery.FunctionInfo
	if err := json.Unmarshal(resp.Result, &list); err != nil {
		t.Fatalf("result: %v", err)
	}
	if len(list) < 2 {
		t.Fatalf("want at least 2 functions, got %d (%v)", len(list), list)
	}
}

func TestHandleFunctions_marshalListError_returnsResponseWithoutResult(t *testing.T) {
	orig := jsonMarshalFunctionsList
	jsonMarshalFunctionsList = func(any) ([]byte, error) { return nil, fmt.Errorf("no") }
	t.Cleanup(func() { jsonMarshalFunctionsList = orig })

	s := testDevServer(t)
	s.functions = map[string]map[string]discovery.FunctionInfo{
		"p": {"F": {Package: "p", Name: "F"}},
	}
	rr := httptest.NewRecorder()
	s.handleFunctions(rr, httptest.NewRequest(http.MethodGet, "/functions", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d", rr.Code)
	}
	var resp DevServerResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success || len(resp.Result) != 0 {
		t.Fatalf("expected success with empty result on marshal failure, got %+v", resp)
	}
}
