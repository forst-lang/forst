package bridgert

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"forst/internal/ftconfig"
)

func TestIntegration_compiledModulesDirSeparateFromBoundary(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}

	bootstrap, err := ResolveBootstrapPath(repoRoot(t), "")
	if err != nil {
		t.Skipf("bootstrap not available: %v", err)
	}
	t.Setenv(envBridgeBootstrap, bootstrap)

	boundaryRoot := t.TempDir()
	modulesDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(modulesDir, "legacy"), 0o755); err != nil {
		t.Fatal(err)
	}
	modulePath := filepath.Join(modulesDir, "legacy", "payment.js")
	if err := os.WriteFile(modulePath, []byte(legacyPaymentAddJS), 0o644); err != nil {
		t.Fatal(err)
	}

	ftconfigPath := filepath.Join(boundaryRoot, "ftconfig.json")
	cfg := ftconfig.Default()
	cfg.Bridge.LegacyModules.Format = ftconfig.LegacyModuleCompiled
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ftconfigPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv(ftconfig.EnvRoot, boundaryRoot)
	t.Setenv(ftconfig.EnvBridgeModulesDir, modulesDir)

	manifest := Manifest{
		Version:      ManifestVersion,
		BoundaryRoot: boundaryRoot,
		Exports: []ExportEntry{
			{ModuleID: compiledLegacyPaymentModuleID, Name: "add", Kind: ExportKindFunction},
		},
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	resetSupervisorForTest()
	MustConfigureFromManifest(string(manifestJSON))

	type addResult struct {
		Sum int `json:"sum"`
	}
	got, err := CallSync[addResult](compiledLegacyPaymentModuleID, "add", 2, 3)
	if err != nil {
		t.Fatalf("CallSync add: %v", err)
	}
	if got.Sum != 5 {
		t.Fatalf("sum = %d want 5", got.Sum)
	}
	if err := Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}
