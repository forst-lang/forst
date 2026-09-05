package invokeserver

import (
	"runtime"
	"testing"
)

func TestApplyListenDefaults_unixOnNonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix default is not used on Windows")
	}
	t.Setenv(envInvokeTransport, "")
	root := t.TempDir()
	cfg := Config{Host: "127.0.0.1", Port: "8081", Runtime: "embedded"}
	ApplyListenDefaults(&cfg, root)
	if cfg.Transport != transportUnix {
		t.Fatalf("Transport = %q, want unix", cfg.Transport)
	}
	want := DefaultInvokeSocketPath(root)
	if cfg.SocketPath != want {
		t.Fatalf("SocketPath = %q, want %q", cfg.SocketPath, want)
	}
	if cfg.BoundaryRoot != root {
		t.Fatalf("BoundaryRoot = %q, want %q", cfg.BoundaryRoot, root)
	}
}

func TestApplyListenDefaults_tcpOptOutViaEnv(t *testing.T) {
	t.Setenv(envInvokeTransport, "tcp")
	root := t.TempDir()
	cfg := Config{Host: "127.0.0.1", Port: "8081", Runtime: "embedded"}
	ApplyListenDefaults(&cfg, root)
	if cfg.Transport != transportTCP {
		t.Fatalf("Transport = %q, want tcp", cfg.Transport)
	}
	if cfg.SocketPath != "" {
		t.Fatalf("SocketPath = %q, want empty", cfg.SocketPath)
	}
}

func TestApplyListenDefaults_explicitUnixFillsSocketPath(t *testing.T) {
	root := t.TempDir()
	cfg := Config{Transport: transportUnix, BoundaryRoot: root}
	ApplyListenDefaults(&cfg, "")
	want := DefaultInvokeSocketPath(root)
	if cfg.SocketPath != want {
		t.Fatalf("SocketPath = %q, want %q", cfg.SocketPath, want)
	}
}
