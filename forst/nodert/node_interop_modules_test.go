package nodert

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIntegration_nestedModuleResolution_realBootstrap(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}

	bootstrap, err := ResolveBootstrapPath(repoRoot(t), "")
	if err != nil {
		t.Skipf("bootstrap not available: %v", err)
	}
	t.Setenv(envNodeBootstrap, bootstrap)

	root := nodeInteropModulesDir(t)
	manifest := Manifest{
		Version:      ManifestVersion,
		BoundaryRoot: root,
		Exports: []ExportEntry{
			{ModuleID: "legacy/api/checkout.ts", Name: "createOrder", Kind: ExportKindFunction},
		},
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	resetSupervisorForTest()
	MustConfigureFromManifest(string(manifestJSON))

	type orderResult struct {
		ID         string  `json:"id"`
		Total      string  `json:"total"`
		TotalCents float64 `json:"totalCents"`
	}
	got, err := CallSync[orderResult]("legacy/api/checkout.ts", "createOrder", 100, "USD")
	if err != nil {
		t.Fatalf("CallSync createOrder: %v", err)
	}
	if got.ID != "ord_1:USD:100.00" {
		t.Fatalf("id = %q", got.ID)
	}
	if got.Total != "USD:110.00" {
		t.Fatalf("total = %q", got.Total)
	}
	if got.TotalCents != 11000 {
		t.Fatalf("totalCents = %v want 11000", got.TotalCents)
	}
	if err := Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func nodeInteropModulesDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(repoRoot(t), "examples", "in", "rfc", "node-interop", "modules")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("modules fixture: %v", err)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}
