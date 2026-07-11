package invokeserver

import (
	"net"
	"strings"
	"testing"

	"forst/internal/discovery"
)

func TestConfig_Addr_embeddedForcesLocalhost(t *testing.T) {
	tests := []struct {
		name string
		host string
	}{
		{"zero", "0.0.0.0"},
		{"empty", ""},
		{"public", "192.168.1.1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{Host: tc.host, Port: "0", Runtime: "embedded"}
			addr := cfg.Addr()
			if !strings.HasPrefix(addr, embeddedListenHost+":") {
				t.Fatalf("Addr() = %q, want prefix %q", addr, embeddedListenHost+":")
			}
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				t.Fatal(err)
			}
			defer ln.Close()
			tcpAddr := ln.Addr().(*net.TCPAddr)
			if !tcpAddr.IP.IsLoopback() {
				t.Fatalf("bound to non-loopback %v", tcpAddr.IP)
			}
		})
	}
}

func TestConfig_BaseURL_embeddedForcesLocalhost(t *testing.T) {
	cfg := Config{Host: "0.0.0.0", Port: "8081", Runtime: "embedded"}
	if got := cfg.BaseURL(); got != "http://127.0.0.1:8081" {
		t.Fatalf("BaseURL() = %q", got)
	}
}

func TestConfig_listenHost_devRuntime(t *testing.T) {
	cfg := Config{Runtime: "dev", Host: ""}
	if got := cfg.Addr(); !strings.HasPrefix(got, embeddedListenHost+":") {
		t.Fatalf("empty host Addr() = %q", got)
	}
	cfg = Config{Runtime: "dev", Host: "0.0.0.0", Port: "9090"}
	if got := cfg.Addr(); got != "0.0.0.0:9090" {
		t.Fatalf("explicit host Addr() = %q", got)
	}
}

func TestConfig_listenPort_defaults(t *testing.T) {
	cfg := Config{Host: "127.0.0.1", Port: "", Runtime: "dev"}
	if got := cfg.Addr(); got != "127.0.0.1:8081" {
		t.Fatalf("Addr() = %q", got)
	}
}

func TestStartAsync_embeddedBindsLoopbackOnly(t *testing.T) {
	backend := &stubBackend{
		functions: map[string]map[string]discovery.FunctionInfo{},
	}
	s := New(Config{Host: "0.0.0.0", Port: "0", Runtime: "embedded"}, backend, DefaultEmbeddedVersion(), nil)
	if err := s.StartAsync(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	if got := s.Config().Addr(); !strings.HasPrefix(got, embeddedListenHost+":") {
		t.Fatalf("Config().Addr() = %q", got)
	}
}
