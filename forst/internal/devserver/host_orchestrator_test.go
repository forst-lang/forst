package devserver

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"forst/internal/compiler"
	"forst/internal/ftconfig"
	"forst/nodert"
)

func TestReload_parentOwnedHostSurvivesGroupKill(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}
	if runtime.GOOS == "windows" {
		t.Skip("unix host sockets")
	}

	root := setupReloadHostFixture(t)
	manifest := nodert.Manifest{
		Version:      nodert.ManifestVersion,
		BoundaryRoot: root,
		Exports: []nodert.ExportEntry{
			{ModuleID: "legacy/counter.ts", Name: "inc", Kind: nodert.ExportKindFunction},
		},
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := ftconfig.LoadFromDir(root)
	if err != nil {
		t.Fatal(err)
	}
	orch := NewHostOrchestrator(nil, root, cfg)
	if err := orch.EnsureRunning(); err != nil {
		t.Fatalf("EnsureRunning: %v", err)
	}
	t.Cleanup(func() { _ = orch.Shutdown() })

	nodePID := nodert.ReadHostMarkerPID(root)
	if nodePID <= 0 {
		t.Fatal("node host pid not recorded")
	}

<<<<<<< Updated upstream
	t.Setenv(nodert.EnvNodeAttachOnly, "1")
	if err := nodert.ConfigureFromManifestForTest(string(manifestJSON)); err != nil {
=======
	t.Setenv(bridgert.EnvBridgeAttachOnly, "1")
	if err := bridgert.ConfigureFromManifestForTest(string(manifestJSON)); err != nil {
>>>>>>> Stashed changes
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = nodert.ShutdownForTest() })
	if _, err := nodert.GetClient(); err != nil {
		t.Fatalf("GetClient attach-only: %v", err)
	}

	goDir := t.TempDir()
	goMain := filepath.Join(goDir, "main.go")
	if err := os.WriteFile(goMain, []byte("package main\nfunc main() { select {} }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	proc, err := compiler.StartGoProgram(goMain, root)
	if err != nil {
		t.Fatal(err)
	}
	if err := proc.Stop(compiler.DefaultGoProgramStopGrace(), compiler.StopOpts{}); err != nil {
		t.Fatal(err)
	}

	hostProc, err := os.FindProcess(nodePID)
	if err != nil {
		t.Fatal(err)
	}
	if err := hostProc.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("node host pid=%d should survive group kill: %v", nodePID, err)
	}

<<<<<<< Updated upstream
	nodert.ResetForTest()
	t.Setenv(nodert.EnvNodeAttachOnly, "1")
	if err := nodert.ConfigureFromManifestForTest(string(manifestJSON)); err != nil {
=======
	bridgert.ResetForTest()
	t.Setenv(bridgert.EnvBridgeAttachOnly, "1")
	if err := bridgert.ConfigureFromManifestForTest(string(manifestJSON)); err != nil {
>>>>>>> Stashed changes
		t.Fatal(err)
	}
	if _, err := nodert.GetClient(); err != nil {
		t.Fatalf("reattach GetClient: %v", err)
	}
	if after := nodert.ReadHostMarkerPID(root); after != nodePID {
		t.Fatalf("node pid changed: %d -> %d", nodePID, after)
	}
	got, err := nodert.CallSyncForTest[float64]("legacy/counter.ts", "inc")
	if err != nil {
		t.Fatalf("CallSync after reattach: %v", err)
	}
	if int(got) != 2 {
		t.Fatalf("inc after reattach = %v want 2", got)
	}
}

func TestHostOrchestrator_shutdownTerminatesSpawnedHost(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix signals")
	}
	// Auth relay restarts a live marker host and spawns Node (needs tsx). This test
	// uses a sleep stand-in and asserts Shutdown terminates the marker pid.
	t.Setenv("FORST_INVOKE_AUTH", "off")

	dir := t.TempDir()
	_, readyPath, err := nodert.ResolveHostSocketPath(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(readyPath), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("sleep", "60")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	payload, err := json.Marshal(map[string]any{
		"pid":    cmd.Process.Pid,
		"socket": strings.TrimSuffix(readyPath, ".ready"),
		"phase":  "app",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(readyPath, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	pid := nodert.ReadHostMarkerPID(dir)
	if pid <= 0 {
		t.Fatalf("marker pid not ready (sleep pid=%d)", cmd.Process.Pid)
	}

	cfg, err := loadHostModeConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	orch := NewHostOrchestrator(nil, dir, cfg)
	if err := orch.EnsureRunning(); err != nil {
		t.Fatal(err)
	}
	if err := orch.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected wait error after orchestrator shutdown")
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("pid=%d still running after orchestrator shutdown", pid)
	}
}

func loadHostModeConfig(root string) (*ftconfig.Config, error) {
	if err := os.WriteFile(filepath.Join(root, "ftconfig.json"), []byte(`{"node":{"hostMode":true,"args":["x"]}}`), 0o644); err != nil {
		return nil, err
	}
	return ftconfig.LoadFromDir(root)
}
