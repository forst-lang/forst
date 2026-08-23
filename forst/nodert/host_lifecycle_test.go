package nodert

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

func TestEnsureHostProcessRunning_restartsLiveHostWhenAuthRelaySet(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix signals")
	}

	dir := t.TempDir()
	socketPath := filepath.Join(dir, ".forst", "node.sock")
	readyPath := filepath.Join(dir, ".forst", "node.sock.ready")
	if err := os.MkdirAll(filepath.Dir(readyPath), 0o755); err != nil {
		t.Fatal(err)
	}

	oldHost := exec.Command("sleep", "60")
	if err := oldHost.Start(); err != nil {
		t.Fatal(err)
	}
	oldPID := oldHost.Process.Pid
	oldDone := make(chan error, 1)
	go func() { oldDone <- oldHost.Wait() }()
	t.Cleanup(func() {
		_ = oldHost.Process.Kill()
		select {
		case <-oldDone:
		case <-time.After(2 * time.Second):
		}
	})

	payload, err := json.Marshal(map[string]any{
		"pid":    oldPID,
		"socket": socketPath,
		"phase":  "app",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(readyPath, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if ReattachSkipReason(readyPath) != "" {
		t.Fatalf("expected live marker, skip=%q", ReattachSkipReason(readyPath))
	}

	relay, err := NewHostInvokeAuthRelay()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = relay.Close() })

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	spawned, proc, err := EnsureHostProcessRunning(HostProcessConfig{
		BoundaryRoot: dir,
		WorkDir:      dir,
		NodePath:     "node",
		ShimArgs:     []string{"missing.mjs"},
		SocketPath:   socketPath,
		ReadyPath:    readyPath,
		AuthRelay:    relay,
		ReadyTimeout: time.Second,
		Log:          log,
	})
	if spawned {
		t.Fatal("expected spawned=false when replacement host cannot become ready")
	}
	if proc != nil {
		t.Fatal("expected nil proc when replacement host fails")
	}
	if err == nil {
		t.Fatal("expected error after restarting for auth handoff")
	}

	select {
	case waitErr := <-oldDone:
		if waitErr == nil {
			t.Fatal("expected old host wait error after terminate")
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("old host pid=%d still running after AuthRelay restart", oldPID)
	}
	if _, statErr := os.Stat(readyPath); !os.IsNotExist(statErr) {
		t.Fatalf("ready marker should be cleaned after failed restart, stat=%v", statErr)
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

func TestWaitForHostMarkerReady_failsFastWhenProcessExits(t *testing.T) {
	exitCh := make(chan error, 1)
	exitCh <- fmt.Errorf("%w: exit status 1", ErrNodeRuntimeDied)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	err := waitForHostMarkerReady(ctx, "/tmp/missing.ready", exitCh, func() string { return "listen EPERM" })
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "host process exited before ready") {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(err.Error(), "listen EPERM") {
		t.Fatalf("stderr not in error: %v", err)
	}
<<<<<<< Updated upstream:forst/nodert/host_lifecycle_test.go
	if !errors.Is(err, ErrNodeRuntimeDied) && !strings.Contains(err.Error(), "node runtime process exited") {
=======
	if !errors.Is(err, ErrBridgeRuntimeDied) && !strings.Contains(err.Error(), "bridge runtime process exited") {
>>>>>>> Stashed changes:forst/bridgert/host_lifecycle_test.go
		t.Fatalf("exit reason not in error: %v", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("wait took %v, expected immediate fail", elapsed)
	}
}
