package invokeserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"forst/internal/discovery"
	"forst/internal/invokedispatch"
)

type stubBackend struct {
	functions map[string]map[string]discovery.FunctionInfo
	invoke    func(pkg, fn string, args json.RawMessage) (*invokedispatch.InvokeResult, error)
}

func (b *stubBackend) Functions() map[string]map[string]discovery.FunctionInfo {
	return b.functions
}

func (b *stubBackend) RefreshFunctions(context.Context) error { return nil }

func (b *stubBackend) Invoke(_ context.Context, pkg, fn string, args json.RawMessage) (*invokedispatch.InvokeResult, error) {
	if b.invoke != nil {
		return b.invoke(pkg, fn, args)
	}
	return &invokedispatch.InvokeResult{Success: true, Result: json.RawMessage(`{"ok":true}`)}, nil
}

func (b *stubBackend) InvokeStream(context.Context, string, string, json.RawMessage) (<-chan invokedispatch.StreamChunk, error) {
	return nil, nil
}

func TestServer_handleHealthAndInvoke(t *testing.T) {
	backend := &stubBackend{
		functions: map[string]map[string]discovery.FunctionInfo{
			"main": {
				"Echo": {Package: "main", Name: "Echo", Runnable: true},
			},
		},
	}
	s := New(Config{Host: "127.0.0.1", Port: "0", Runtime: "embedded"}, backend, DefaultEmbeddedVersion(), nil)

	rr := httptest.NewRecorder()
	s.HandleHealth(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("health: %d", rr.Code)
	}

	body := bytes.NewBufferString(`{"package":"main","function":"Echo","args":[{"message":"hi"}]}`)
	rr2 := httptest.NewRecorder()
	s.HandleInvoke(rr2, httptest.NewRequest(http.MethodPost, "/invoke", body))
	if rr2.Code != http.StatusOK {
		t.Fatalf("invoke: %d %s", rr2.Code, rr2.Body.String())
	}
	var resp Response
	if err := json.NewDecoder(rr2.Body).Decode(&resp); err != nil || !resp.Success {
		t.Fatalf("resp: %+v err=%v", resp, err)
	}
}

func TestRegistryBackend_invokeHandler(t *testing.T) {
	reg := invokedispatch.NewRegistry()
	reg.Register(discovery.FunctionInfo{
		Package: "main",
		Name:    "Echo",
		Runnable: true,
	}, func(args json.RawMessage) (any, error) {
		return map[string]string{"echo": "hi"}, nil
	})
	backend := NewRegistryBackend(reg)
	result, err := backend.Invoke(nil, "main", "Echo", json.RawMessage(`[]`))
	if err != nil || !result.Success {
		t.Fatalf("invoke: %+v err=%v", result, err)
	}
}
