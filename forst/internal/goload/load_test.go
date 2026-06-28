package goload

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/packages"
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
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module ghostmod\n\ngo 1.23\n"), 0o644); err != nil {
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
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testmod\n\ngo 1.23\n"), 0o644); err != nil {
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

