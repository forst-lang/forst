package forstpkg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildForstPackageImportPaths(t *testing.T) {
	root := filepath.Clean("/proj")
	paths := map[string][]string{
		"catalog": {filepath.Join(root, "catalog", "catalog.ft")},
		"orders":  {filepath.Join(root, "orders", "orders.ft")},
		"empty":   {},
	}
	got := BuildForstPackageImportPaths(root, "example.com/app", paths)
	if got["example.com/app/catalog"] != "catalog" {
		t.Fatalf("catalog path: %v", got)
	}
	if got["example.com/app/orders"] != "orders" {
		t.Fatalf("orders path: %v", got)
	}
	if _, ok := got["example.com/app/empty"]; ok {
		t.Fatalf("empty file list should be skipped: %v", got)
	}
}

func TestBuildForstPackageImportPaths_rootPackage(t *testing.T) {
	root := filepath.Clean("/proj")
	paths := map[string][]string{
		"main": {filepath.Join(root, "main.ft")},
	}
	got := BuildForstPackageImportPaths(root, "example.com/app", paths)
	if got["example.com/app"] != "main" {
		t.Fatalf("root package import path: %v", got)
	}
}

func TestBuildForstPackageImportPaths_packageNamedFileInParentDir(t *testing.T) {
	root := filepath.Clean("/proj")
	paths := map[string][]string{
		"version": {filepath.Join(root, "internal", "version.ft")},
	}
	got := BuildForstPackageImportPaths(root, "example.com/app", paths)
	if got["example.com/app/internal"] != "version" {
		t.Fatalf("dir import path: %v", got)
	}
	if got["example.com/app/internal/version"] != "version" {
		t.Fatalf("package-named file import path: %v", got)
	}
}

func TestBuildForstPackageImportPaths_skipsEmptyFileList(t *testing.T) {
	got := BuildForstPackageImportPaths("/proj", "example.com/app", map[string][]string{
		"empty": {},
	})
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %v", got)
	}
}

func TestImportPathForDir_rootAndSubdir(t *testing.T) {
	root := filepath.Clean("/proj")
	got, err := ImportPathForDir(root, "example.com/app", root)
	if err != nil {
		t.Fatal(err)
	}
	if got != "example.com/app" {
		t.Fatalf("root: got %q", got)
	}
	got, err = ImportPathForDir(root, "example.com/app", filepath.Join(root, "catalog"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "example.com/app/catalog" {
		t.Fatalf("subdir: got %q", got)
	}
}

func TestImportPathForDir_relError(t *testing.T) {
	orig := buildImportPathsRel
	buildImportPathsRel = func(string, string) (string, error) { return "", os.ErrInvalid }
	t.Cleanup(func() { buildImportPathsRel = orig })

	_, err := ImportPathForDir("/proj", "example.com/app", "/proj/pkg")
	if err == nil {
		t.Fatal("expected rel error")
	}
}

func TestBuildForstPackageImportPaths_skipsRelError(t *testing.T) {
	orig := buildImportPathsRel
	buildImportPathsRel = func(string, string) (string, error) { return "", os.ErrInvalid }
	t.Cleanup(func() { buildImportPathsRel = orig })

	got := BuildForstPackageImportPaths("/proj", "example.com/app", map[string][]string{
		"pkg": {filepath.Join("/proj", "pkg", "file.ft")},
	})
	if len(got) != 0 {
		t.Fatalf("expected rel error to skip entry, got %v", got)
	}
}
