package nodert

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestIntegration_CallAsync_realBootstrap(t *testing.T) {
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
	tsFile := filepath.Join(legacyDir, "payment.ts")
	if err := os.WriteFile(tsFile, []byte(`export async function create(amount: number, currency: string): Promise<{ id: string; amount: number }> {
  return { id: "pay_async", amount };
}
export async function concurrentEcho(n: number): Promise<{ echo: number }> {
  await new Promise((resolve) => setTimeout(resolve, 5));
  return { echo: n };
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest := Manifest{
		Version:      ManifestVersion,
		BoundaryRoot: root,
		Exports: []ExportEntry{
			{ModuleID: "legacy/payment.ts", Name: "create", Kind: ExportKindAsyncFunction},
			{ModuleID: "legacy/payment.ts", Name: "concurrentEcho", Kind: ExportKindAsyncFunction},
		},
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	resetSupervisorForTest()
	MustConfigureFromManifest(string(manifestJSON))

	type createResult struct {
		ID     string  `json:"id"`
		Amount float64 `json:"amount"`
	}
	got, err := CallAsync[createResult]("legacy/payment.ts", "create", 100, "USD")
	if err != nil {
		t.Fatalf("CallAsync create: %v", err)
	}
	if got.ID != "pay_async" || got.Amount != 100 {
		t.Fatalf("create = %#v", got)
	}

	type echoResult struct {
		Echo int `json:"echo"`
	}
	echo, err := CallAsync[echoResult]("legacy/payment.ts", "concurrentEcho", 42)
	if err != nil {
		t.Fatalf("CallAsync concurrentEcho: %v", err)
	}
	if echo.Echo != 42 {
		t.Fatalf("echo = %#v", echo)
	}
	if err := Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func TestIntegration_CallSync_realBootstrap(t *testing.T) {
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
	tsFile := filepath.Join(legacyDir, "payment.ts")
	if err := os.WriteFile(tsFile, []byte(`export function add(a: number, b: number): { sum: number } {
  return { sum: a + b };
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest := Manifest{
		Version:      ManifestVersion,
		BoundaryRoot: root,
		Exports: []ExportEntry{
			{ModuleID: "legacy/payment.ts", Name: "add", Kind: ExportKindFunction},
		},
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	resetSupervisorForTest()
	MustConfigureFromManifest(string(manifestJSON))

	type result struct {
		Sum float64 `json:"sum"`
	}
	got, err := CallSync[result]("legacy/payment.ts", "add", 40, 2)
	if err != nil {
		t.Fatalf("CallSync: %v", err)
	}
	if got.Sum != 42 {
		t.Fatalf("sum = %v want 42", got.Sum)
	}
	if err := Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
