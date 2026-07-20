package nodert

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestBootstrap_nodeDeath_failFastAndRespawn(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}

	bootstrap, err := ResolveBootstrapPath(repoRoot(t), "")
	if err != nil {
		t.Skipf("bootstrap not available: %v", err)
	}
	t.Setenv(envNodeBootstrap, bootstrap)

	root := t.TempDir()
	legacyDir := filepath.Join(root, "legacy")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tsFile := filepath.Join(legacyDir, "slow.ts")
	if err := os.WriteFile(tsFile, []byte(`export async function hang(): Promise<{ ok: boolean }> {
  await new Promise(() => {});
  return { ok: true };
}
export function add(a: number, b: number): { sum: number } {
  return { sum: a + b };
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest := Manifest{
		Version:      ManifestVersion,
		BoundaryRoot: root,
		Exports: []ExportEntry{
			{ModuleID: "legacy/slow.ts", Name: "hang", Kind: ExportKindAsyncFunction},
			{ModuleID: "legacy/slow.ts", Name: "add", Kind: ExportKindFunction},
		},
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	resetSupervisorForTest()
	t.Cleanup(resetSupervisorForTest)
	if err := configureFromManifest(string(manifestJSON)); err != nil {
		t.Fatal(err)
	}

	type sumResult struct {
		Sum float64 `json:"sum"`
	}
	got, err := CallSync[sumResult]("legacy/slow.ts", "add", 1, 1)
	if err != nil {
		t.Fatalf("first CallSync: %v", err)
	}
	if got.Sum != 2 {
		t.Fatalf("sum = %v want 2", got.Sum)
	}

	supervisorMu.Lock()
	proc := supervisorInst.proc
	supervisorMu.Unlock()
	if proc == nil || proc.cmd == nil || proc.cmd.Process == nil {
		t.Fatal("expected supervised bootstrap process")
	}
	deadPID := proc.cmd.Process.Pid
	socketPath, readyPath, err := ResolveBootstrapSocketPath(root)
	if err != nil {
		t.Fatal(err)
	}

	hangErr := make(chan error, 1)
	go func() {
		_, err := CallAsync[map[string]bool]("legacy/slow.ts", "hang")
		hangErr <- err
	}()

	time.Sleep(100 * time.Millisecond)
	if err := proc.cmd.Process.Kill(); err != nil {
		t.Fatalf("kill node: %v", err)
	}
	_ = proc.wait()

	select {
	case err := <-hangErr:
		if err == nil {
			t.Fatal("expected in-flight call to fail after node death")
		}
		if !errors.Is(err, ErrNodeRuntimeDied) {
			t.Fatalf("in-flight err = %v want ErrNodeRuntimeDied", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("in-flight call did not fail after node death")
	}

	got2, err := CallSync[sumResult]("legacy/slow.ts", "add", 40, 2)
	if err != nil {
		t.Fatalf("respawn CallSync: %v", err)
	}
	if got2.Sum != 42 {
		t.Fatalf("respawn sum = %v want 42", got2.Sum)
	}

	marker, ok := readHostReadyMarker(readyPath)
	if !ok {
		t.Fatalf("missing ready marker after respawn: %s", readyPath)
	}
	if marker.PID == deadPID {
		t.Fatalf("ready marker pid = %d still references killed process", deadPID)
	}
	if _, err := os.Stat(socketPath); err != nil {
		t.Fatalf("socket missing after respawn: %v", err)
	}

	supervisorMu.Lock()
	respawnProc := supervisorInst.proc
	supervisorMu.Unlock()
	if respawnProc == nil || respawnProc.cmd == nil || respawnProc.cmd.Process == nil {
		t.Fatal("expected respawned bootstrap process")
	}
	if respawnProc.cmd.Process.Pid == deadPID {
		t.Fatalf("respawn pid = %d same as killed pid", deadPID)
	}
	client, err := GetClient()
	if err != nil {
		t.Fatalf("GetClient after respawn: %v", err)
	}
	if err := client.Ping(); err != nil {
		t.Fatalf("Ping after respawn: %v", err)
	}

	if err := Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func TestDefaultCallTimeout_isRequestSafe(t *testing.T) {
	if DefaultCallTimeout != 30*time.Second {
		t.Fatalf("DefaultCallTimeout = %s want 30s", DefaultCallTimeout)
	}
}
