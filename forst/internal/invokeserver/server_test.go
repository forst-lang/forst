package invokeserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"forst/internal/discovery"
	"forst/internal/invokedispatch"
)

type stubBackend struct {
	functions    map[string]map[string]discovery.FunctionInfo
	refreshErr   error
	invoke       func(pkg, fn string, args json.RawMessage) (*invokedispatch.InvokeResult, error)
	invokeStream func(ctx context.Context, pkg, fn string, args json.RawMessage) (<-chan invokedispatch.StreamChunk, error)
}

func (b *stubBackend) Functions() map[string]map[string]discovery.FunctionInfo {
	return b.functions
}

func (b *stubBackend) RefreshFunctions(context.Context) error {
	return b.refreshErr
}

func (b *stubBackend) Invoke(ctx context.Context, pkg, fn string, args json.RawMessage) (*invokedispatch.InvokeResult, error) {
	if b.invoke != nil {
		return b.invoke(pkg, fn, args)
	}
	return &invokedispatch.InvokeResult{Success: true, Result: json.RawMessage(`{"ok":true}`)}, nil
}

func (b *stubBackend) InvokeStream(ctx context.Context, pkg, fn string, args json.RawMessage) (<-chan invokedispatch.StreamChunk, error) {
	if b.invokeStream != nil {
		return b.invokeStream(ctx, pkg, fn, args)
	}
	return nil, nil
}

type stubLogger struct {
	errors []string
}

func (l *stubLogger) Infof(string, ...any)  {}
func (l *stubLogger) Debugf(string, ...any) {}
func (l *stubLogger) Errorf(format string, args ...any) {
	l.errors = append(l.errors, fmt.Sprintf(format, args...))
}

func newTestServer(t *testing.T, backend DispatchBackend, opts ...func(*Config)) *Server {
	t.Helper()
	cfg := Config{Host: "127.0.0.1", Port: "0", Runtime: "embedded"}
	for _, opt := range opts {
		opt(&cfg)
	}
	return New(cfg, backend, DefaultEmbeddedVersion(), nil)
}

type failReader struct{}

func (failReader) Read([]byte) (int, error) { return 0, fmt.Errorf("read failed") }

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

func pollHealthOK(t *testing.T, baseURL string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server at %s did not become healthy", baseURL)
}

func TestServer_handleHealthAndInvoke(t *testing.T) {
	backend := &stubBackend{
		functions: map[string]map[string]discovery.FunctionInfo{
			"main": {
				"Echo": {Package: "main", Name: "Echo", Runnable: true},
			},
		},
	}
	s := newTestServer(t, backend)

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
		Package:  "main",
		Name:     "Echo",
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

var _ = io.Discard
