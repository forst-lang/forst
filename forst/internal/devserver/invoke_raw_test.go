package devserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"forst/internal/discovery"
	"forst/internal/executor"
)

func TestHandleInvokeRaw_wrongMethod(t *testing.T) {
	s := testDevServer(t)
	rr := httptest.NewRecorder()
	s.handleInvokeRaw(rr, httptest.NewRequest(http.MethodGet, "/invoke/raw", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET /invoke/raw: %d", rr.Code)
	}
}

func TestHandleInvokeRaw_missingQueryParams(t *testing.T) {
	s := testDevServer(t)
	rr := httptest.NewRecorder()
	s.handleInvokeRaw(rr, httptest.NewRequest(http.MethodPost, "/invoke/raw", strings.NewReader(`[]`)))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleInvokeRaw_emptyBody(t *testing.T) {
	s := testDevServer(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/invoke/raw?package=p&function=f", strings.NewReader(""))
	s.handleInvokeRaw(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleInvokeRaw_invalidJSONBody(t *testing.T) {
	s := testDevServer(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/invoke/raw?package=p&function=f", strings.NewReader(`not json`))
	s.handleInvokeRaw(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleInvokeRaw_notJSONArray(t *testing.T) {
	s := testDevServer(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/invoke/raw?package=p&function=f", strings.NewReader(`{}`))
	s.handleInvokeRaw(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleInvokeRaw_packageNotFound(t *testing.T) {
	s := testDevServer(t)
	s.functions = make(map[string]map[string]discovery.FunctionInfo)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/invoke/raw?package=missing&function=Fn", strings.NewReader(`[]`))
	s.handleInvokeRaw(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleInvokeRaw_success_matchesInvoke(t *testing.T) {
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
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/invoke/raw?package=mypkg&function=Fn", strings.NewReader(`[]`))
	s.handleInvokeRaw(rr, req)
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
