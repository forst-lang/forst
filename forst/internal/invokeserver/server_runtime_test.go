package invokeserver

import (
	"fmt"
	"net"
	"testing"
	"time"

	"forst/internal/discovery"
)

func TestSetBackend_BackendFunctions(t *testing.T) {
	s := newTestServer(t, &stubBackend{
		functions: map[string]map[string]discovery.FunctionInfo{
			"main": {"Fn": {Package: "main", Name: "Fn"}},
		},
	})
	if got := s.BackendFunctions()["main"]["Fn"].Name; got != "Fn" {
		t.Fatalf("BackendFunctions = %q", got)
	}

	s.SetBackend(nil)
	if s.BackendFunctions() != nil {
		t.Fatal("expected nil backend functions")
	}
}

func TestEffectiveTimeouts_defaults(t *testing.T) {
	s := New(Config{}, &stubBackend{}, DefaultEmbeddedVersion(), nil)
	read, write := s.effectiveTimeouts()
	if read != 30*time.Second || write != 30*time.Second {
		t.Fatalf("timeouts = %v %v", read, write)
	}
}

func TestStartAsync_refreshFailure_logs(t *testing.T) {
	log := &stubLogger{}
	backend := &stubBackend{refreshErr: fmt.Errorf("refresh failed")}
	s := New(Config{Host: "127.0.0.1", Port: "0", Runtime: "embedded"}, backend, DefaultEmbeddedVersion(), log)
	if err := s.StartAsync(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Stop() }()
	if len(log.errors) == 0 {
		t.Fatal("expected refresh error log")
	}
}

func TestServer_StartOnMux_servesRoutes(t *testing.T) {
	backend := &stubBackend{}
	s := New(Config{Host: "127.0.0.1", Port: "0", Runtime: "embedded"}, backend, DefaultEmbeddedVersion(), nil)
	if err := s.StartAsync(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Stop() })
	pollHealthOK(t, "http://"+s.BoundAddr())
}

func TestServer_Start_servesRoutes(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	backend := &stubBackend{}
	s := New(Config{Host: "127.0.0.1", Port: fmt.Sprintf("%d", port), Runtime: "embedded"}, backend, DefaultEmbeddedVersion(), nil)
	go func() { _ = s.Start() }()
	t.Cleanup(func() { _ = s.Stop() })
	pollHealthOK(t, fmt.Sprintf("http://127.0.0.1:%d", port))
}

func TestStartAsync_invalidPort(t *testing.T) {
	s := New(Config{Host: "127.0.0.1", Port: "invalid-port", Runtime: "embedded"}, &stubBackend{}, DefaultEmbeddedVersion(), nil)
	if err := s.StartAsync(); err == nil {
		t.Fatal("expected listen error")
	}
}

func TestStop_neverStarted(t *testing.T) {
	s := newTestServer(t, &stubBackend{})
	if err := s.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
