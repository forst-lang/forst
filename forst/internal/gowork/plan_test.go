package gowork

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoWork_childEnv_stripsInheritedGOWORK(t *testing.T) {
	base := []string{"GOWORK=/parent/go.work", "PATH=/bin"}
	env := ChildEnv(base, LinkPlan{Mode: LinkReplace}, "/app")
	foundOff := false
	for _, e := range env {
		if e == "GOWORK=off" {
			foundOff = true
		}
		if strings.HasPrefix(e, "GOWORK=") && e != "GOWORK=off" {
			t.Fatalf("inherited GOWORK leaked: %s", e)
		}
	}
	if !foundOff {
		t.Fatal("expected GOWORK=off in child env")
	}
}

func TestGoWork_planForRun_usesReplaceInTempModule(t *testing.T) {
	dir := t.TempDir()
	plan, err := PlanForRun(dir, filepath.Join(dir, ".forst", "run", "s1"), true)
	if err != nil {
		// OK when FORST_GOMOD_ROOT unset in test env
		t.Skip("compiler module not found:", err)
	}
	if plan.Mode != LinkReplace {
		t.Fatalf("want LinkReplace, got %v", plan.Mode)
	}
}

func TestGoWork_writeTestGoMod_hasPerPackageReplaces(t *testing.T) {
	dir := t.TempDir()
	modDir := filepath.Join(dir, "mod")
	path := filepath.Join(modDir, "go.mod")
	testDir := filepath.Join(dir, "test", "api")
	libDir := filepath.Join(dir, "lib", "auth")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatal(err)
	}
	replaces := []PackageReplace{
		{ImportPath: "demo/api", Dir: testDir},
		{ImportPath: "demo/auth", Dir: libDir},
	}
	if err := WriteTestGoMod(path, "/compiler/forst", replaces, "demo/api"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "replace demo/api => ../test/api") {
		t.Fatalf("missing api replace: %s", s)
	}
	if !strings.Contains(s, "replace demo/auth => ../lib/auth") {
		t.Fatalf("missing auth replace: %s", s)
	}
}

func TestGoWork_writeRunGoMod_hasReplaceForst(t *testing.T) {
	dir := t.TempDir()
	compilerDir := filepath.Join(dir, "compiler", "forst")
	if err := os.MkdirAll(compilerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	userDir := filepath.Join(dir, "app")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sandbox := filepath.Join(dir, "sandbox")
	if err := os.MkdirAll(sandbox, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sandbox, "go.mod")
	link := ForstRuntimeLink{ReplaceDir: compilerDir}
	if err := WriteRunGoMod(path, link, "example.com/app", userDir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "replace forst => ../compiler/forst") {
		t.Fatalf("missing relative replace forst: %s", s)
	}
	if !strings.Contains(s, "replace example.com/app => ../app") {
		t.Fatalf("missing relative user module replace: %s", s)
	}
}

func TestGoWork_writeRunGoMod_emptyLink_errors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "go.mod")
	if err := WriteRunGoMod(path, ForstRuntimeLink{}, "example.com/app", t.TempDir()); err == nil {
		t.Fatal("expected error for empty forst link")
	}
}

func TestGoWork_resolveForstRuntimeLink_fromUserGoMod(t *testing.T) {
	root := t.TempDir()
	forstGomod := filepath.Join(root, ".forst-gomod")
	if err := os.MkdirAll(forstGomod, 0o755); err != nil {
		t.Fatal(err)
	}
	forstTarget := filepath.Join(root, "forst-runtime")
	if err := os.MkdirAll(forstTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(forstGomod, "go.mod"), []byte("module app\n\nreplace forst => ./forst-runtime\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link, err := ResolveForstRuntimeLink(root)
	if err != nil {
		t.Fatal(err)
	}
	if link.ReplaceDir != filepath.Clean(forstTarget) {
		t.Fatalf("ReplaceDir = %q want %q", link.ReplaceDir, forstTarget)
	}
}
