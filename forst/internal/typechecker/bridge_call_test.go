package typechecker

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/bridgeinterop"
	"forst/internal/testutil"
)

func TestBridgeExportParamTypes_nilResolverErrors(t *testing.T) {
	tc := New(nil, false)
	_, err := tc.BridgeExportParamTypes("legacy/payment.ts", "create")
	if err == nil || !strings.Contains(err.Error(), "node index resolver not initialized") {
		t.Fatalf("got %v", err)
	}
}

func TestBridgeCallTarget_nilTypeChecker(t *testing.T) {
	var tc *TypeChecker
	_, ok := tc.BridgeCallTarget("payment", "create")
	if ok {
		t.Fatal("expected false for nil typechecker")
	}
}

func TestBridgeCallTarget_resolvesSyncExport(t *testing.T) {
	root := t.TempDir()
	writeNodeFixture(t, root)

	src := `package main
import "./legacy/payment" js

func main() {
	payment.create(10.0, "usd")
}
`
	tc, _ := MustTypecheck(t, src, testutil.TypecheckOpts{
		NodeBoundaryRoot: root,
		ForstFileDir:     root,
	})
	target, ok := tc.BridgeCallTarget("payment", "create")
	if !ok {
		t.Fatal("expected BridgeCallTarget to resolve")
	}
	if target.ModuleID != "legacy/payment.ts" || target.ExportName != "create" {
		t.Fatalf("target = %+v", target)
	}
	if target.Kind != bridgeinterop.ExportKindAsyncFunction {
		t.Fatalf("kind = %q", target.Kind)
	}
}

func TestBridgeCallTarget_compiledManifestUsesJSModuleID(t *testing.T) {
	if _, err := exec.LookPath("esbuild"); err != nil {
		t.Skip("esbuild not on PATH")
	}

	root := t.TempDir()
	legacyDir := filepath.Join(root, "legacy")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tsFile := filepath.Join(legacyDir, "payment.ts")
	if err := os.WriteFile(tsFile, []byte(`export async function create(amount: number, currency: string): Promise<{ id: string }> {
  return { id: "x" };
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ftconfig.json"), []byte(`{"bridge":{"legacyModules":{"format":"compiled"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	src := `package main
import "./legacy/payment" js

func main() {
	payment.create(10.0, "usd")
}
`
	tc, _ := MustTypecheck(t, src, testutil.TypecheckOpts{
		NodeBoundaryRoot: root,
		ForstFileDir:     root,
	})
	target, ok := tc.BridgeCallTarget("payment", "create")
	if !ok {
		t.Fatal("expected BridgeCallTarget to resolve")
	}
	if target.ModuleID != "legacy/payment.js" {
		t.Fatalf("target.ModuleID = %q want legacy/payment.js", target.ModuleID)
	}
	if target.SourceModuleID != "legacy/payment.ts" {
		t.Fatalf("target.SourceModuleID = %q want legacy/payment.ts", target.SourceModuleID)
	}
	state := tc.BridgeRuntimeState()
	if len(state.Manifest.Exports) != 1 || state.Manifest.Exports[0].ModuleID != "legacy/payment.js" {
		t.Fatalf("manifest exports = %+v", state.Manifest.Exports)
	}
	if _, err := os.Stat(filepath.Join(root, ".forst", "js", "legacy", "payment.js")); err != nil {
		t.Fatalf("precompiled module: %v", err)
	}
}

func TestBridgeExportParamTypes_returnsIndexSignature(t *testing.T) {
	root := t.TempDir()
	writeNodeFixture(t, root)

	src := `package main
import "./legacy/payment" js

func main() {
	payment.create(10.0, "usd")
}
`
	tc, _ := MustTypecheck(t, src, testutil.TypecheckOpts{
		NodeBoundaryRoot: root,
		ForstFileDir:     root,
	})
	params, err := tc.BridgeExportParamTypes("legacy/payment.ts", "create")
	if err != nil {
		t.Fatal(err)
	}
	if len(params) != 2 {
		t.Fatalf("params = %v", params)
	}
}

func TestBridgeExportParamTypes_missingExportErrors(t *testing.T) {
	root := t.TempDir()
	writeNodeFixture(t, root)

	src := `package main
import "./legacy/payment" js

func main() {}
`
	tc, _ := MustTypecheck(t, src, testutil.TypecheckOpts{
		NodeBoundaryRoot: root,
		ForstFileDir:     root,
	})
	_, err := tc.BridgeExportParamTypes("legacy/payment.ts", "missing")
	if err == nil {
		t.Fatal("expected error for missing export")
	}
}

func TestBridgeModuleForLocal_resolvesAlias(t *testing.T) {
	root := t.TempDir()
	writeNodeFixture(t, root)

	src := `package main
import "./legacy/payment" js

func main() {}
`
	tc, _ := MustTypecheck(t, src, testutil.TypecheckOpts{
		NodeBoundaryRoot: root,
		ForstFileDir:     root,
	})
	moduleID, ok := tc.BridgeModuleForLocal("payment")
	if !ok || moduleID != "legacy/payment.ts" {
		t.Fatalf("moduleID = %q ok = %v", moduleID, ok)
	}
}
