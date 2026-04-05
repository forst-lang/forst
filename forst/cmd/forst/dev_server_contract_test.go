package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Normative HTTP contract: examples/in/rfc/typescript-client/02-forst-dev-http-contract.md

func TestDevServer_healthGET_matchesContract(t *testing.T) {
	t.Parallel()
	s := testDevServer(t)
	rr := httptest.NewRecorder()
	s.handleHealth(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d body: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type: want application/json, got %q", ct)
	}
	var resp DevServerResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success || resp.Output != "Forst HTTP server is healthy" {
		t.Fatalf("response: %+v", resp)
	}
}

func TestDevServer_methodNotAllowed_returnsJSONEnvelope(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		handler func(*DevServer, http.ResponseWriter, *http.Request)
		method  string
		path    string
	}{
		{"health", (*DevServer).handleHealth, http.MethodPost, "/health"},
		{"functions", (*DevServer).handleFunctions, http.MethodPost, "/functions"},
		{"invoke", (*DevServer).handleInvoke, http.MethodGet, "/invoke"},
		{"types", (*DevServer).handleTypes, http.MethodPost, "/types"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := testDevServer(t)
			if tc.name == "types" {
				s.typesGenerator = NewTypeScriptGenerator(s.log)
			}
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, http.NoBody)
			tc.handler(s, rr, req)
			if rr.Code != http.StatusMethodNotAllowed {
				t.Fatalf("want 405, got %d: %s", rr.Code, rr.Body.String())
			}
			if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
				t.Fatalf("Content-Type: want application/json, got %q", ct)
			}
			var resp DevServerResponse
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatal(err)
			}
			if resp.Success || resp.Error == "" {
				t.Fatalf("expected success=false and error set, got %+v", resp)
			}
		})
	}
}

func TestDevServer_typesGET_JSONEnvelope_notRawTypeScriptFile(t *testing.T) {
	t.Parallel()
	s := testDevServer(t)
	s.typesGenerator = NewTypeScriptGenerator(s.log)
	rr := httptest.NewRecorder()
	s.handleTypes(rr, httptest.NewRequest(http.MethodGet, "/types", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /types: %d %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type: want application/json, got %q", ct)
	}
	if disp := rr.Header().Get("Content-Disposition"); disp != "" {
		t.Fatalf("unexpected Content-Disposition %q (contract: parse JSON envelope)", disp)
	}
	var resp DevServerResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success || resp.Output == "" {
		t.Fatalf("expected success and non-empty output, got %+v", resp)
	}
}
