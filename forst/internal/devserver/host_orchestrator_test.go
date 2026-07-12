package devserver

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

	t.Setenv(nodert.EnvNodeAttachOnly, "1")
	if err := nodert.ConfigureFromManifestForTest(string(manifestJSON)); err != nil {
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

	nodert.ResetForTest()
	t.Setenv(nodert.EnvNodeAttachOnly, "1")
	if err := nodert.ConfigureFromManifestForTest(string(manifestJSON)); err != nil {
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

	dir := t.TempDir()
	readyPath := filepath.Join(dir, ".forst", "node.sock.ready")
	if err := os.MkdirAll(filepath.Dir(readyPath), 0o755); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		cmd := exec.Command("sleep", "60")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := cmd.Start(); err != nil {
			close(done)
			return
		}
		payload, _ := json.Marshal(map[string]any{
			"pid":    cmd.Process.Pid,
			"socket": filepath.Join(dir, ".forst", "node.sock"),
			"phase":  "app",
		})
		_ = os.WriteFile(readyPath, payload, 0o644)
		_ = cmd.Wait()
		close(done)
	}()

	deadline := time.Now().Add(2 * time.Second)
	var pid int
	for time.Now().Before(deadline) {
		pid = nodert.ReadHostMarkerPID(dir)
		if pid > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if pid <= 0 {
		t.Fatal("marker pid not ready")
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
	proc, err := os.FindProcess(pid)
	if err != nil {
		t.Fatal(err)
	}
	if err := proc.Signal(syscall.Signal(0)); err == nil {
		t.Fatalf("pid=%d should be terminated after orchestrator shutdown", pid)
	}
	<-done
}

func loadHostModeConfig(root string) (*ftconfig.Config, error) {
	if err := os.WriteFile(filepath.Join(root, "ftconfig.json"), []byte(`{"node":{"hostMode":true,"args":["x"]}}`), 0o644); err != nil {
		return nil, err
	}
	return ftconfig.LoadFromDir(root)
}
