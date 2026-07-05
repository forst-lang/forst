package modulecheck_test

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/modulecheck"
	"forst/internal/testmod"
)

func TestCheckModuleProviders_twoPhaseMultiPackageModule(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), testmod.GoModContent("sibling_demo"))
	alphaDir := filepath.Join(dir, "alpha")
	betaDir := filepath.Join(dir, "beta")
	for _, d := range []string{alphaDir, betaDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(alphaDir, "types.ft"), `package alpha

type Config = {
	Version: String,
}
`)
	writeFile(t, filepath.Join(betaDir, "use.ft"), `package beta

import "sibling_demo/alpha"

func Use(cfg: alpha.Config) {
	println(cfg.Version)
}

func main() {
	Use({ Version: "1.0" })
}
`)
	result, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{ModuleRoot: dir})
	if err != nil {
		t.Fatalf("CheckModuleProviders: %v", err)
	}
	if result.ForstPackageTypeChecker("beta") == nil {
		t.Fatalf("missing beta typechecker; packages=%v", result.ForstPkgToFiles)
	}
	if result.ForstPackageTypeChecker("alpha") == nil {
		t.Fatalf("missing alpha typechecker; packages=%v", result.ForstPkgToFiles)
	}
}

func TestCheckModuleProviders_crossPkg(t *testing.T) {
	root := "../../../examples/in/rfc/providers/cross_pkg"
	result, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{ModuleRoot: root})
	if err != nil {
		t.Fatalf("CheckModuleProviders: %v", err)
	}
	beta := result.ForstPackageTypeChecker("beta")
	if beta == nil {
		t.Fatal("missing beta tc")
	}
	slots := beta.FunctionProviders["Handle"]
	if len(slots) != 1 || slots[0].RootIdent != "Logger" {
		t.Fatalf("Handle providers = %v", slots)
	}
	alpha := result.ForstPackageTypeChecker("alpha")
	if alpha == nil {
		t.Fatal("missing alpha tc")
	}
	logSlots := alpha.FunctionProviders["LogExpiry"]
	if len(logSlots) != 1 {
		t.Fatalf("LogExpiry providers = %v", logSlots)
	}
	if path, ok := beta.ImportPathForLocal("alpha"); !ok || path != "providers_cross_pkg_demo/alpha" {
		t.Fatalf("beta import path for alpha = %q ok=%v", path, ok)
	}
	if got := result.ImportPathForForstPackage("beta"); got != "providers_cross_pkg_demo/beta" {
		t.Fatalf("ImportPathForForstPackage(beta) = %q", got)
	}
}

func TestCheckModuleProviders_crossPkg_missingWiringAtRoot(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), testmod.GoModContent("cross_neg"))
	alphaDir := filepath.Join(dir, "alpha")
	betaDir := filepath.Join(dir, "beta")
	for _, d := range []string{alphaDir, betaDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(alphaDir, "log.ft"), `package alpha

type Logger = { Info(msg String) }

func LogExpiry(id String) {
	use logger: Logger
	logger.Info(id)
}
`)
	writeFile(t, filepath.Join(betaDir, "handle.ft"), `package beta

import "cross_neg/alpha"

func Handle(id String) {
	alpha.LogExpiry(id)
}
`)
	writeFile(t, filepath.Join(betaDir, "handle_test.ft"), `package beta

import "testing"

func TestHandle(t *testing.T) {
	Handle("tok")
}
`)
	_, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{ModuleRoot: dir})
	if err == nil {
		t.Fatal("expected wiring root error")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckModuleProviders_mainWiringRoot(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples", "in", "rfc", "providers")
	result, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{ModuleRoot: root})
	if err != nil {
		t.Fatalf("CheckModuleProviders: %v", err)
	}
	tc := result.ForstPackageTypeChecker("providers_demo")
	if tc == nil {
		t.Fatal("missing providers_demo tc")
	}
	if len(tc.FunctionProviders["mainWiringDemo"]) != 0 {
		t.Fatalf("mainWiringDemo should be runnable, providers = %v", tc.FunctionProviders["mainWiringDemo"])
	}
}
