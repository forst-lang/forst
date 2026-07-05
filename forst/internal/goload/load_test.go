package goload

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/testmod"

	"golang.org/x/tools/go/packages"
	"go/types"
	"strings"
)

func moduleRootFromWD(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found from cwd")
		}
		dir = parent
	}
}

func TestLoadByPkgPath_fmt(t *testing.T) {
	dir := moduleRootFromWD(t)
	m, err := LoadByPkgPath(dir, []string{"fmt"})
	if err != nil {
		t.Fatal(err)
	}
	p := m["fmt"]
	if p == nil || p.Types == nil {
		t.Fatalf("fmt package missing: %#v", m)
	}
	if p.Types.Scope().Lookup("Println") == nil {
		t.Fatal("fmt.Println not in scope")
	}
}

func TestImportPathFromForst(t *testing.T) {
	if got := ImportPathFromForst(`"net/http"`); got != "net/http" {
		t.Fatalf("got %q", got)
	}
}

func TestLoadByPkgPath_emptyPaths(t *testing.T) {
	t.Parallel()
	m, err := LoadByPkgPath(t.TempDir(), nil)
	if err != nil || m != nil {
		t.Fatalf("empty import list: m=%v err=%v", m, err)
	}
}

func TestLoadByPkgPath_strconv(t *testing.T) {
	dir := moduleRootFromWD(t)
	m, err := LoadByPkgPath(dir, []string{"strconv"})
	if err != nil {
		t.Fatal(err)
	}
	p := m["strconv"]
	if p == nil || p.Types == nil {
		t.Fatalf("strconv package missing: %#v", m)
	}
	if p.Types.Scope().Lookup("Atoi") == nil {
		t.Fatal("strconv.Atoi not in scope")
	}
}

func TestImportPathFromForst_whitespaceAndQuotes(t *testing.T) {
	t.Parallel()
	if got := ImportPathFromForst(`  "encoding/json"  `); got != "encoding/json" {
		t.Fatalf("got %q", got)
	}
}

func TestLoadByPkgPath_os(t *testing.T) {
	dir := moduleRootFromWD(t)
	m, err := LoadByPkgPath(dir, []string{"os"})
	if err != nil {
		t.Fatal(err)
	}
	p := m["os"]
	if p == nil || p.Types == nil {
		t.Fatalf("os package missing: %#v", m)
	}
	if p.Types.Scope().Lookup("Getenv") == nil {
		t.Fatal("os.Getenv not in scope")
	}
}

func TestLoadByPkgPath_cacheHit(t *testing.T) {
	dir := moduleRootFromWD(t)
	m1, err := LoadByPkgPath(dir, []string{"fmt"})
	if err != nil {
		t.Fatal(err)
	}
	m2, err := LoadByPkgPath(dir, []string{"fmt"})
	if err != nil {
		t.Fatal(err)
	}
	if m1["fmt"] != m2["fmt"] {
		t.Fatal("expected cached fmt package pointer")
	}
}

func TestPackageLoadOK(t *testing.T) {
	t.Parallel()
	if PackageLoadOK(nil, "fmt") {
		t.Fatal("nil package")
	}
}

func TestPackageLoadOK_acceptsStdlib(t *testing.T) {
	dir := moduleRootFromWD(t)
	m, err := LoadByPkgPath(dir, []string{"fmt"})
	if err != nil {
		t.Fatal(err)
	}
	if !PackageLoadOK(m["fmt"], "fmt") {
		t.Fatal("expected fmt to pass PackageLoadOK")
	}
}

func TestPackageLoadOK_rejectsGhostWithoutSources(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("ghostmod")), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	ClearLoadCacheForTest()
	cfg := &packages.Config{Mode: packages.NeedTypes, Dir: dir, Env: loadPackagesEnv(dir)}
	pkgs, err := packages.Load(cfg, "ghostmod/sub")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pkgs {
		if packageLoadOKAt(p, "ghostmod/sub", dir) {
			t.Fatal("ghost package without Go sources should not pass packageLoadOKAt")
		}
	}
}

func TestLoadByPkgPath_normalizesSubdir(t *testing.T) {
	root := moduleRootFromWD(t)
	sub := filepath.Join(root, "internal", "goload")
	m, err := LoadByPkgPath(sub, []string{"fmt"})
	if err != nil {
		t.Fatal(err)
	}
	if m["fmt"] == nil {
		t.Fatal("expected fmt when Dir is a module subdir")
	}
}

func TestLoadByPkgPath_singlePathCacheFromBatch(t *testing.T) {
	dir := moduleRootFromWD(t)
	ClearLoadCacheForTest()
	batch, err := LoadByPkgPath(dir, []string{"fmt", "strings"})
	if err != nil {
		t.Fatal(err)
	}
	single, err := LoadByPkgPath(dir, []string{"fmt"})
	if err != nil {
		t.Fatal(err)
	}
	if batch["fmt"] != single["fmt"] {
		t.Fatal("expected single-path load to reuse batch cache entry")
	}
}

func TestLoadByPkgPath_modulePackageWithoutGoSources(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("testmod")), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	ClearLoadCacheForTest()
	m, err := LoadByPkgPath(dir, []string{"testmod/sub"})
	if err == nil && m["testmod/sub"] != nil {
		t.Fatalf("expected unresolved package omitted, got %#v", m)
	}
	// Mixed load: stdlib succeeds, module subpackage without Go sources is omitted.
	ClearLoadCacheForTest()
	m, err = LoadByPkgPath(dir, []string{"testing", "testmod/sub"})
	if err != nil {
		t.Fatal(err)
	}
	if m["testing"] == nil || m["testing"].Types == nil {
		t.Fatal("expected testing package")
	}
	if m["testmod/sub"] != nil {
		t.Fatal("expected testmod/sub omitted when directory has no Go files")
	}
}

func TestPackageHasGoSources(t *testing.T) {
	t.Parallel()
	if packageHasGoSources(nil) {
		t.Fatal("nil package")
	}
	if packageHasGoSources(&packages.Package{}) {
		t.Fatal("empty package")
	}
	p := &packages.Package{GoFiles: []string{"a.go"}}
	if !packageHasGoSources(p) {
		t.Fatal("expected GoFiles to count")
	}
	p = &packages.Package{CompiledGoFiles: []string{"a.go"}}
	if !packageHasGoSources(p) {
		t.Fatal("expected CompiledGoFiles to count")
	}
}

func TestPackageLoadOKAt_rejectsWrongPath(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	m, err := LoadByPkgPath(dir, []string{"fmt"})
	if err != nil {
		t.Fatal(err)
	}
	if packageLoadOKAt(m["fmt"], "strings", dir) {
		t.Fatal("expected path mismatch to fail")
	}
}

func TestImportPathFromForst_barePath(t *testing.T) {
	t.Parallel()
	if got := ImportPathFromForst("fmt"); got != "fmt" {
		t.Fatalf("got %q", got)
	}
}

func TestLoadByPkgPath_cachesErrors(t *testing.T) {
	dir := moduleRootFromWD(t)
	ClearLoadCacheForTest()
	_, err1 := LoadByPkgPath(dir, []string{"forst/nonexistent/pkg/xyz"})
	_, err2 := LoadByPkgPath(dir, []string{"forst/nonexistent/pkg/xyz"})
	if err1 == nil {
		t.Skip("loader returned success unexpectedly")
	}
	if err2 == nil || err1.Error() != err2.Error() {
		t.Fatalf("expected cached error, err1=%v err2=%v", err1, err2)
	}
}

func TestModulePath_missingGoModReturnsEmpty(t *testing.T) {
	t.Parallel()
	if got := ModulePath(t.TempDir()); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestModulePath_goModWithoutModuleLine(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("go 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ModulePath(dir); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestModuleRootWithGoMod_fromFilePath(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module testmod\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "pkg", "x.go")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ModuleRootWithGoMod(file)
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Fatalf("got %q want %q", got, root)
	}
}

func TestPackageLoadOKAt_acceptsModuleSubpackageWithGoSources(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("loadok")), 0o644); err != nil {
		t.Fatal(err)
	}
	p := &packages.Package{
		Types:   mustLoadokSubTypes(t),
		GoFiles: []string{filepath.Join(dir, "sub", "sub.go")},
	}
	if !packageLoadOKAt(p, "loadok/sub", dir) {
		t.Fatal("expected module subpackage with Go sources to pass")
	}
}

func TestPackageLoadOKAt_acceptsTypedPackageWithoutGoFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("loadok")), 0o644); err != nil {
		t.Fatal(err)
	}
	p := &packages.Package{
		Types: mustLoadokSubTypes(t),
	}
	if !packageLoadOKAt(p, "loadok/sub", dir) {
		t.Fatal("expected typed in-module package without GoFiles to pass")
	}
	emptyName := types.NewPackage("loadok/sub", "")
	p2 := &packages.Package{Types: emptyName}
	if packageLoadOKAt(p2, "loadok/sub", dir) {
		t.Fatal("expected empty Types.Name to fail")
	}
}

func mustLoadokSubTypes(t *testing.T) *types.Package {
	t.Helper()
	return types.NewPackage("loadok/sub", "sub")
}

func TestLoadByPkgPath_packagesLoadError(t *testing.T) {
	dir := moduleRootFromWD(t)
	ClearLoadCacheForTest()
	_, err := LoadByPkgPath(dir, []string{"\x00invalid/path"})
	if err == nil {
		t.Fatal("expected load error")
	}
}

func TestStoreLoadCache_skipsEmptyPathKey(t *testing.T) {
	dir := moduleRootFromWD(t)
	ClearLoadCacheForTest()
	p := &packages.Package{Types: mustLoadokSubTypes(t)}
	storeLoadCache(dir, []string{"fmt", "strings"}, map[string]*packages.Package{
		"fmt":     p,
		"strings": p,
		"":        p,
	}, nil)
	single, err := LoadByPkgPath(dir, []string{"fmt"})
	if err != nil {
		t.Fatal(err)
	}
	if single["fmt"] == nil {
		t.Fatal("expected fmt cached despite empty path key in batch")
	}
}

func TestLoadByPkgPath_noTypedPackagesWithoutMessages(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("ghostonly")), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	ClearLoadCacheForTest()
	_, err := LoadByPkgPath(dir, []string{"ghostonly/sub"})
	if err == nil {
		t.Fatal("expected error when no typed packages resolve")
	}
	if !strings.Contains(err.Error(), "no typed packages") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadByPkgPathUncached_noTypedPackagesWithoutErrorMessages(t *testing.T) {
	orig := packagesLoadFn
	packagesLoadFn = func(*packages.Config, ...string) ([]*packages.Package, error) {
		return []*packages.Package{{ID: "ghost"}}, nil
	}
	t.Cleanup(func() { packagesLoadFn = orig })

	_, err := loadByPkgPathUncached(t.TempDir(), []string{"ghost"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no typed packages for [ghost]") {
		t.Fatalf("unexpected error: %v", err)
	}
}

