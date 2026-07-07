package testutil

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModuleRoot_findsGoMod(t *testing.T) {
	root := ModuleRoot(t)
	if !strings.HasSuffix(root, "forst") && !strings.Contains(root, "forst") {
		t.Fatalf("unexpected module root: %s", root)
	}
}

func TestModuleRoot_goModNotFound(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	msg := stubFail(t, func() {
		ModuleRoot(t)
	})
	if !strings.Contains(msg, "go.mod not found") {
		t.Fatalf("msg = %q", msg)
	}
}

func TestModuleRoot_getwdError(t *testing.T) {
	orig := moduleRootGetwd
	t.Cleanup(func() { moduleRootGetwd = orig })
	moduleRootGetwd = func() (string, error) {
		return "", errors.New("getwd failed")
	}
	msg := stubFail(t, func() {
		ModuleRoot(t)
	})
	if !strings.Contains(msg, "getwd failed") {
		t.Fatalf("msg = %q", msg)
	}
}

func TestModuleRootFrom_success(t *testing.T) {
	root := ModuleRoot(t)
	sub := filepath.Join(root, "internal", "testutil")
	got := ModuleRootFrom(t, sub)
	if got != root {
		t.Fatalf("ModuleRootFrom = %q, want %q", got, root)
	}
}

func TestModuleRootFrom_notFound(t *testing.T) {
	orig := moduleRootFromFind
	t.Cleanup(func() { moduleRootFromFind = orig })
	moduleRootFromFind = func(string) string { return "" }
	msg := stubFailf(t, func() {
		ModuleRootFrom(t, t.TempDir())
	})
	if !strings.Contains(msg, "module root not found") {
		t.Fatalf("msg = %q", msg)
	}
}

func TestExamplePath(t *testing.T) {
	path := ExamplePath(t, "basic.ft")
	if !strings.HasSuffix(path, "basic.ft") {
		t.Fatalf("unexpected path: %s", path)
	}
}

func TestExamplePath_notFound(t *testing.T) {
	msg := stubFailf(t, func() {
		ExamplePath(t, "no_such_example_ever.ft")
	})
	if !strings.Contains(msg, "example file not found") {
		t.Fatalf("msg = %q", msg)
	}
}

func TestExamplePath_absError(t *testing.T) {
	orig := examplePathAbs
	t.Cleanup(func() { examplePathAbs = orig })
	examplePathAbs = func(string) (string, error) {
		return "", errors.New("abs failed")
	}
	msg := stubFail(t, func() {
		ExamplePath(t, "basic.ft")
	})
	if !strings.Contains(msg, "abs failed") {
		t.Fatalf("msg = %q", msg)
	}
}
