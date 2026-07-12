package nodert

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"forst/internal/ftconfig"

	logrus "github.com/sirupsen/logrus"
)

func TestEnsureHostProcessRunning_idempotentWhenMarkerLive(t *testing.T) {
	dir := t.TempDir()
	readyPath := filepath.Join(dir, ".forst", "node.sock.ready")
	if err := os.MkdirAll(filepath.Dir(readyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]any{
		"pid":    os.Getpid(),
		"socket": filepath.Join(dir, ".forst", "node.sock"),
		"phase":  "app",
	})
	if err := os.WriteFile(readyPath, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := HostProcessConfig{
		BoundaryRoot: dir,
		WorkDir:      dir,
		NodePath:     "node",
		ShimArgs:     []string{"missing.mjs"},
		SocketPath:   filepath.Join(dir, ".forst", "node.sock"),
		ReadyPath:    readyPath,
		Log:          logrus.New(),
	}
	spawned, proc, err := EnsureHostProcessRunning(cfg)
	if err != nil {
		t.Fatalf("EnsureHostProcessRunning: %v", err)
	}
	if spawned {
		t.Fatal("expected spawned=false when marker live")
	}
	if proc != nil {
		t.Fatal("expected nil proc when marker live")
	}
}

func TestHostProcessConfigFromFTConfig_requiresHostModeArgs(t *testing.T) {
	_, err := HostProcessConfigFromFTConfig(&ftconfig.Config{
		Node: ftconfig.NodeConfig{HostMode: true},
	}, t.TempDir(), logrus.New())
	if err == nil {
		t.Fatal("expected error for empty node.args")
	}
}

func TestTerminateHostPID_terminatesSleepChild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix signals")
	}

	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	if err := TerminateHostPID(pid, 500*time.Millisecond); err != nil {
		t.Fatalf("TerminateHostPID: %v", err)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected wait error after terminate")
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("pid=%d still running after TerminateHostPID", pid)
	}
}
