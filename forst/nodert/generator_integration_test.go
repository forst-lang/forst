package nodert

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIntegration_Generators_realBootstrap(t *testing.T) {
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
	tsFile := filepath.Join(legacyDir, "generators.ts")
	if err := os.WriteFile(tsFile, []byte(`export function* syncNumbers(limit: number): Generator<number> {
  for (let i = 0; i < limit; i++) yield i;
}
export async function* asyncNumbers(limit: number): AsyncGenerator<number> {
  for (let i = 0; i < limit; i++) yield i;
}
export function* withFinally(): Generator<number> {
  try { yield 1; yield 2; yield 3; } finally { (globalThis as any).__finally = true; }
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest := Manifest{
		Version:      ManifestVersion,
		BoundaryRoot: root,
		Exports: []ExportEntry{
			{ModuleID: "legacy/generators.ts", Name: "syncNumbers", Kind: ExportKindGenerator},
			{ModuleID: "legacy/generators.ts", Name: "asyncNumbers", Kind: ExportKindAsyncGenerator},
			{ModuleID: "legacy/generators.ts", Name: "withFinally", Kind: ExportKindGenerator},
		},
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	resetSupervisorForTest()
	MustConfigureFromManifest(string(manifestJSON))

	var syncSum float64
	it, err := OpenGen[float64]("legacy/generators.ts", "syncNumbers", 3)
	if err != nil {
		t.Fatalf("OpenGen: %v", err)
	}
	defer func() { _ = it.Close() }()
	for {
		step, err := it.Next()
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if step.Kind == GenStepDone {
			break
		}
		if step.Kind == GenStepYield {
			syncSum += step.Value
		}
	}
	if syncSum != 3 {
		t.Fatalf("syncSum = %v want 3", syncSum)
	}

	ait, err := OpenAsyncGen[float64]("legacy/generators.ts", "asyncNumbers", 3)
	if err != nil {
		t.Fatalf("OpenAsyncGen: %v", err)
	}
	defer func() { _ = ait.Close() }()
	var asyncSum float64
	for {
		step, err := ait.Next()
		if err != nil {
			t.Fatalf("async Next: %v", err)
		}
		if step.Kind == GenStepDone {
			break
		}
		if step.Kind == GenStepYield {
			asyncSum += step.Value
		}
	}
	if asyncSum != 3 {
		t.Fatalf("asyncSum = %v want 3", asyncSum)
	}

	fit, err := OpenGen[float64]("legacy/generators.ts", "withFinally")
	if err != nil {
		t.Fatalf("OpenGen withFinally: %v", err)
	}
	for {
		step, err := fit.Next()
		if err != nil {
			t.Fatalf("withFinally Next: %v", err)
		}
		if step.Kind == GenStepDone {
			break
		}
		if step.Kind == GenStepYield && step.Value == 2 {
			break
		}
	}
	if err := fit.Close(); err != nil {
		t.Fatalf("withFinally Close: %v", err)
	}

	if err := Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}
