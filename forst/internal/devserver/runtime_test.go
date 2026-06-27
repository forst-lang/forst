package devserver

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"forst/internal/discovery"

	"github.com/sirupsen/logrus"
)

func TestDevServer_Stop_nilServerNoop(t *testing.T) {
	s := &DevServer{}
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
}

func TestDevServer_Stop_nonNilServerClosePath(t *testing.T) {
	s := &DevServer{server: &http.Server{}}
	if err := s.Stop(); err != nil {
		t.Fatalf("expected nil close error for empty server, got %v", err)
	}
}

func TestDevServer_Start_invalidPortReturnsError_andInitializesTypesGenerator(t *testing.T) {
	s := testDevServer(t)
	s.port = "invalid-port"
	err := s.Start()
	if err == nil {
		t.Fatal("expected start error for invalid port")
	}
	if s.typesGenerator == nil {
		t.Fatal("expected types generator initialization before listen failure")
	}
}

func TestDevServer_logStartupInfo_includesEndpoints(t *testing.T) {
	s := testDevServer(t)
	buf := &bytes.Buffer{}
	s.log.SetOutput(buf)
	s.port = "8080"

	s.logStartupInfo()

	output := buf.String()
	for _, fragment := range []string{
		"HTTP server listening on port 8080",
		"GET  /functions",
		"POST /invoke",
		"POST /invoke/raw",
		"GET  /types",
		"GET  /health",
		"GET  /version",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("startup log missing %q in output:\n%s", fragment, output)
		}
	}
}

func TestDevServer_discoverFunctions_successUpdatesCache(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	rootDir := t.TempDir()
	cfg := &mockConfig{}

	s := &DevServer{
		log:        log,
		discoverer: discovery.NewDiscoverer(rootDir, log, cfg),
		functions: map[string]map[string]discovery.FunctionInfo{
			"stale": {
				"Old": {Package: "stale", Name: "Old"},
			},
		},
	}

	if err := s.discoverFunctions(); err != nil {
		t.Fatalf("discoverFunctions success path returned error: %v", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.functions == nil {
		t.Fatal("expected functions map to be set")
	}
	if _, ok := s.functions["stale"]; ok {
		t.Fatalf("expected stale map to be replaced by fresh discovery result, got %+v", s.functions)
	}
}

func TestDevServer_Start_logsWarningWhenInitialDiscoveryFails(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	s := testDevServer(t)
	s.log = log
	s.port = "invalid-port"
	s.discoverer = discovery.NewDiscoverer(t.TempDir(), log, nil)

	err := s.Start()
	if err == nil {
		t.Fatal("expected listen error due to invalid port")
	}
	if s.typesGenerator == nil {
		t.Fatal("expected types generator to be initialized even when discovery fails")
	}
}

func TestDevServer_discoverFunctions_errorPropagates(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	s := &DevServer{
		log:        log,
		discoverer: discovery.NewDiscoverer(t.TempDir(), log, nil),
	}

	err := s.discoverFunctions()
	if err == nil {
		t.Fatal("expected discoverFunctions error")
	}
	if !strings.Contains(err.Error(), "failed to discover functions") {
		t.Fatalf("unexpected error: %v", err)
	}
}
