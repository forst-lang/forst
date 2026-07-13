package invokeserver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"forst/internal/ftconfig"
)

func TestShouldStartEmbedded(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		env     string
		want    bool
	}{
		{"ftconfig true", true, "", true},
		{"env 1", false, "1", true},
		{"env true", false, "true", true},
		{"disabled", false, "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(envInvokeEnabled, tc.env)
			if got := shouldStartEmbedded(tc.enabled); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestEffectivePort(t *testing.T) {
	t.Setenv(envInvokePort, "9999")
	if got := effectivePort(ftconfig.ServerConfig{Port: "8081"}); got != "9999" {
		t.Fatalf("port = %q", got)
	}
	t.Setenv(envInvokePort, "")
	if got := effectivePort(ftconfig.ServerConfig{Embedded: true}); got != ftconfig.DefaultEmbeddedInvokePort {
		t.Fatalf("default port = %q", got)
	}
}

func TestResolveBoundaryRoot_env(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envBoundaryRoot, dir)
	got, err := resolveBoundaryRoot()
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Fatalf("root = %q", got)
	}
}

func TestResolveBoundaryRoot_walksAncestor(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ftconfig.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envBoundaryRoot, "")
	t.Chdir(sub)
	got, err := resolveBoundaryRoot()
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Fatalf("root = %q want %q", got, root)
	}
}

func TestWriteInvokeReady(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{Host: "127.0.0.1", Port: "8081", Runtime: "embedded"}
	if err := writeInvokeReady(dir, cfg); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, ".forst", "invoke.ready"))
	if err != nil {
		t.Fatal(err)
	}
	var payload InvokeReadyPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.URL != "http://127.0.0.1:8081" || payload.Runtime != "embedded" {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestWriteInvokeReady_invalidPath(t *testing.T) {
	err := writeInvokeReady("../evil", Config{})
	if err == nil || !strings.Contains(err.Error(), "invalid ready path") {
		t.Fatalf("err = %v", err)
	}
}

func TestEmbeddedRuntime_start_disabled(t *testing.T) {
	rt := newEmbeddedRuntime(embeddedDeps{
		resolveRoot: func() (string, error) { return t.TempDir(), nil },
		loadConfig: func(string) (*ftconfig.Config, error) {
			return &ftconfig.Config{}, nil
		},
		writeReady: func(string, Config) error { t.Fatal("should not write"); return nil },
		newServer:  New,
	})
	if err := rt.Start(); err != nil {
		t.Fatal(err)
	}
	if rt.server != nil {
		t.Fatal("expected no server when disabled")
	}
}

func TestEmbeddedRuntime_start_enabled(t *testing.T) {
	workDir := t.TempDir()
	rt := newEmbeddedRuntime(embeddedDeps{
		resolveRoot: func() (string, error) { return workDir, nil },
		loadConfig: func(string) (*ftconfig.Config, error) {
			return &ftconfig.Config{Server: ftconfig.ServerConfig{Embedded: true, Port: "0"}}, nil
		},
		writeReady: writeInvokeReady,
		newServer:  New,
	})
	if err := rt.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if rt.server != nil {
			_ = rt.server.Stop()
		}
	})
	if rt.server == nil {
		t.Fatal("expected server")
	}
	raw, err := os.ReadFile(filepath.Join(workDir, ".forst", "invoke.ready"))
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		t.Fatal("empty ready file")
	}
}

func TestEmbeddedRuntime_start_listenError(t *testing.T) {
	rt := newEmbeddedRuntime(embeddedDeps{
		resolveRoot: func() (string, error) { return t.TempDir(), nil },
		loadConfig: func(string) (*ftconfig.Config, error) {
			return &ftconfig.Config{Server: ftconfig.ServerConfig{Embedded: true, Port: "invalid-port"}}, nil
		},
		writeReady: writeInvokeReady,
		newServer:  New,
	})
	err := rt.Start()
	if err == nil {
		t.Fatal("expected listen error")
	}
}

func TestGlobalRegistry_singleton(t *testing.T) {
	r1 := GlobalRegistry()
	r2 := GlobalRegistry()
	if r1 != r2 {
		t.Fatal("expected same registry instance")
	}
}

func TestMustStartEmbedded_idempotent(t *testing.T) {
	workDir := t.TempDir()
	t.Setenv(envBoundaryRoot, workDir)
	if err := os.WriteFile(filepath.Join(workDir, "ftconfig.json"), []byte(`{"server":{"embedded":true,"port":"0"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	MustStartEmbedded()
	t.Cleanup(func() {
		if defaultRuntime.server != nil {
			_ = defaultRuntime.server.Stop()
		}
	})
	MustStartEmbedded()
	if defaultRuntime.server == nil {
		t.Fatal("expected server started")
	}
}

func TestWaitForShutdown_notifyShutdown(t *testing.T) {
	rt := &EmbeddedRuntime{}
	ready := make(chan struct{})
	done := make(chan struct{})
	go func() {
		rt.WaitForShutdown(func(shutdown <-chan struct{}) {
			close(ready)
			<-shutdown
		})
		close(done)
	}()
	<-ready
	rt.NotifyShutdown()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("WaitForShutdown did not unblock")
	}
}

func TestWaitForShutdown_stopsServer(t *testing.T) {
	workDir := t.TempDir()
	rt := newEmbeddedRuntime(embeddedDeps{
		resolveRoot: func() (string, error) { return workDir, nil },
		loadConfig: func(string) (*ftconfig.Config, error) {
			return &ftconfig.Config{Server: ftconfig.ServerConfig{Embedded: true, Port: "0"}}, nil
		},
		writeReady: writeInvokeReady,
		newServer:  New,
	})
	if err := rt.Start(); err != nil {
		t.Fatal(err)
	}
	if rt.server == nil {
		t.Fatal("expected server")
	}
	ready := make(chan struct{})
	done := make(chan struct{})
	go func() {
		rt.WaitForShutdown(func(shutdown <-chan struct{}) {
			close(ready)
			<-shutdown
		})
		close(done)
	}()
	<-ready
	rt.NotifyShutdown()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("WaitForShutdown did not return")
	}
}

func TestNotifyShutdown_idempotent(t *testing.T) {
	rt := &EmbeddedRuntime{}
	rt.NotifyShutdown()
	rt.NotifyShutdown()
	ch := rt.shutdownCh()
	if ch == nil {
		t.Fatal("expected new shutdown channel after idempotent notify")
	}
}

func TestPackageNotifyShutdown_unblocksWaitForShutdown(t *testing.T) {
	done := make(chan struct{})
	go func() {
		WaitForShutdown()
		close(done)
	}()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		NotifyShutdown()
		select {
		case <-done:
			return
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
	t.Fatal("package WaitForShutdown did not return after NotifyShutdown")
}
