package invokeserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"forst/internal/discovery"
)

type captureLogger struct {
	infos  []string
	errors []string
}

func (l *captureLogger) Infof(format string, args ...any) {
	l.infos = append(l.infos, fmt.Sprintf(format, args...))
}

func (l *captureLogger) Debugf(string, ...any) {}

func (l *captureLogger) Errorf(format string, args ...any) {
	l.errors = append(l.errors, fmt.Sprintf(format, args...))
}

func TestLoggingMiddleware_logsHealthRequest(t *testing.T) {
	log := &captureLogger{}
	s := New(Config{Host: "127.0.0.1", Port: "0", Runtime: "embedded"}, &stubBackend{
		functions: map[string]map[string]discovery.FunctionInfo{},
	}, DefaultEmbeddedVersion(), log)

	mux := http.NewServeMux()
	s.RegisterRoutes(mux)
	handler := s.loggingMiddleware(mux)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if len(log.infos) != 1 {
		t.Fatalf("infos = %v", log.infos)
	}
	if !strings.Contains(log.infos[0], "GET /health 200") {
		t.Fatalf("unexpected log: %q", log.infos[0])
	}
}

func TestHandleInvoke_logsFunctionCall(t *testing.T) {
	log := &captureLogger{}
	s := New(Config{Host: "127.0.0.1", Port: "0", Runtime: "embedded"}, &stubBackend{
		functions: map[string]map[string]discovery.FunctionInfo{
			"mypkg": {"Fn": {}},
		},
	}, DefaultEmbeddedVersion(), log)

	rr := httptest.NewRecorder()
	body := strings.NewReader(`{"package":"mypkg","function":"Fn","args":[]}`)
	handler := s.loggingMiddleware(http.HandlerFunc(s.handleInvoke))
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/invoke", body))

	if len(log.infos) < 2 {
		t.Fatalf("infos = %v", log.infos)
	}
	if !strings.Contains(log.infos[0], "call mypkg.Fn streaming=false") {
		t.Fatalf("missing call log: %v", log.infos)
	}
	if !strings.Contains(log.infos[1], "POST /invoke") {
		t.Fatalf("missing http log: %v", log.infos)
	}
}
