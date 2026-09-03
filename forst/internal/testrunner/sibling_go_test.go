package testrunner

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/testmod"
)

// Tip handoff 03: forst test must include same-package hand-written .go (HelperAdd).
func TestRun_mixedPackage_siblingGoLinked(t *testing.T) {
	if testing.Short() {
		t.Skip("runs real go test")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("mixedprobe")), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgDir := filepath.Join(dir, "domain")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "helper.go"), []byte(`package domain

func HelperAdd(a, b int) int { return a + b }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "add.ft"), []byte(`package domain

func Sum(a Int, b Int): Int {
	return HelperAdd(a, b)
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "add_test.ft"), []byte(`package domain

import "testing"

func TestSum(t *testing.T) {
	if Sum(2, 3) != 5 {
		t.Fatal("sum")
	}
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	code, err := Run(Options{
		ModuleRoot:   dir,
		BoundaryRoot: dir,
		Paths:        []string{"./domain"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if code != ExitSuccess {
		t.Fatalf("expected success, got %d", code)
	}
}

func TestRun_goOnlyDep_replaceToRealDir(t *testing.T) {
	if testing.Short() {
		t.Skip("runs real go test")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("goprobe")), 0o644); err != nil {
		t.Fatal(err)
	}
	utilDir := filepath.Join(dir, "internal", "cryptoutil")
	appDir := filepath.Join(dir, "app")
	for _, d := range []string{utilDir, appDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(utilDir, "util.go"), []byte(`package cryptoutil

func Tag() string { return "ok" }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "app.ft"), []byte(`package app

import "goprobe/internal/cryptoutil"

func Label(): String {
	return cryptoutil.Tag()
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "app_test.ft"), []byte(`package app

import "testing"

func TestLabel(t *testing.T) {
	if Label() != "ok" {
		t.Fatal("label")
	}
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	code, err := Run(Options{
		ModuleRoot:   dir,
		BoundaryRoot: dir,
		Paths:        []string{"./app"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if code != ExitSuccess {
		t.Fatalf("expected success, got %d", code)
	}
}
