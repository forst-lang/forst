package modulecheck

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/testmod"
	"forst/internal/testutil"
)

func TestModuleResult_nilSafeAccessors(t *testing.T) {
	t.Parallel()
	var r *ModuleResult
	if r.ImportPathToForstPkg() != nil {
		t.Fatal("nil ImportPathToForstPkg should return nil")
	}
	if r.ForstPackageTypeChecker("x") != nil {
		t.Fatal("nil ForstPackageTypeChecker should return nil")
	}
}

func TestCheckModuleProviders_nestedProbeModule(t *testing.T) {
	root, _ := testutil.WriteProbeModuleFixture(t, true)
	_, err := CheckModuleProviders(nil, Options{ModuleRoot: root})
	if err != nil {
		t.Fatalf("modulecheck: %v", err)
	}
}

func TestFindForstFiles_skipsVendorGitNodeModules(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, dir := range []string{".git", "vendor", "node_modules"} {
		p := filepath.Join(root, dir, "hidden.ft")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("package main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	keep := filepath.Join(root, "keep.ft")
	if err := os.WriteFile(keep, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := findForstFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || filepath.Base(got[0]) != "keep.ft" {
		t.Fatalf("got %v", got)
	}
}

func TestFindForstFiles_skipsSkipFtSuffix(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "ok.ft"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "broken.skip.ft"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := findForstFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || filepath.Base(got[0]) != "ok.ft" {
		t.Fatalf("got %v", got)
	}
}

func TestFindForstFiles_skipsNestedGoMod(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "go.mod"), []byte(testmod.GoModContent("nestedmod")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "skip.ft"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "root.ft"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := findForstFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || filepath.Base(got[0]) != "root.ft" {
		t.Fatalf("got %v", got)
	}
}

func TestFindForstFiles_skipsNestedFtconfig(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	nested := filepath.Join(root, "rfc", "node-interop")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "ftconfig.json"), []byte(`{"node":{"enabled":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "main.ft"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "root.ft"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := findForstFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || filepath.Base(got[0]) != "root.ft" {
		t.Fatalf("got %v", got)
	}
}

func TestCloneSlots_emptyReturnsEmptyMap(t *testing.T) {
	t.Parallel()
	got := cloneSlots(nil)
	if len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}

func TestCheckModuleProviders_packageFilter(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("filtmod")), 0o644); err != nil {
		t.Fatal(err)
	}
	catalogDir := filepath.Join(dir, "catalog")
	if err := os.MkdirAll(catalogDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeModuleFile(t, filepath.Join(catalogDir, "a.ft"), `package catalog

func A() {}
`)
	writeModuleFile(t, filepath.Join(dir, "orders.ft"), `package orders

func B() {}
`)
	result, err := CheckModuleProviders(nil, Options{
		ModuleRoot:    dir,
		PackageFilter: map[string]struct{}{"catalog": {}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ForstPackageTypeChecker("orders") != nil {
		t.Fatal("orders should be filtered out")
	}
	if result.ForstPackageTypeChecker("catalog") == nil {
		t.Fatal("catalog should be included")
	}
}

func TestCheckModuleProviders_skipsUnparseableFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("skipmod")), 0o644); err != nil {
		t.Fatal(err)
	}
	writeModuleFile(t, filepath.Join(dir, "good.ft"), `package main

func main() {}
`)
	writeModuleFile(t, filepath.Join(dir, "bad.ft"), `not valid {{{`)
	result, err := CheckModuleProviders(nil, Options{ModuleRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.ForstPkgToFiles) != 1 {
		t.Fatalf("expected only good.ft package, got %v", result.ForstPkgToFiles)
	}
}

func TestCheckModuleProviders_typecheckErrorReturnsResult(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("errmod")), 0o644); err != nil {
		t.Fatal(err)
	}
	writeModuleFile(t, filepath.Join(dir, "bad.ft"), `package main

func Broken(): String {
	return 1
}
`)
	_, err := CheckModuleProviders(nil, Options{ModuleRoot: dir})
	if err == nil {
		t.Fatal("expected typecheck error")
	}
}

func TestCheckModuleProviders_missingRootErrors(t *testing.T) {
	t.Parallel()
	_, err := CheckModuleProviders(nil, Options{
		ModuleRoot: filepath.Join(t.TempDir(), "missing-subdir"),
	})
	if err == nil {
		t.Fatal("expected findForstFiles error")
	}
}

func TestCheckModuleProviders_revalidateDeferredWiringErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("revmod")), 0o644); err != nil {
		t.Fatal(err)
	}
	writeModuleFile(t, filepath.Join(dir, "host.ft"), `package host

import "testing"

type Logger = { info(msg String) }
type NopLogger = {}

func (NopLogger) info(msg String) {}

func TestHost(t *testing.T) {
	with { BadKey: &NopLogger {} } {
	}
}
`)
	_, err := CheckModuleProviders(nil, Options{ModuleRoot: dir})
	if err == nil {
		t.Fatal("expected deferred wiring revalidation error")
	}
}

func TestFindForstFiles_walkPermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can read chmod 000 directories")
	}
	root := t.TempDir()
	secret := filepath.Join(root, "secret")
	if err := os.Mkdir(secret, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(secret, 0o755) })
	_, err := findForstFiles(root)
	if err == nil {
		t.Fatal("expected walk error for unreadable directory")
	}
}

func writeModuleFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
