package gowork

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"forst/internal/goload"
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
	if err := WriteRunGoMod(path, link, "example.com/app", userDir, false); err != nil {
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
	if err := WriteRunGoMod(path, ForstRuntimeLink{}, "example.com/app", t.TempDir(), false); err == nil {
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
	if err := os.WriteFile(filepath.Join(forstTarget, "go.mod"), []byte("module forst\n\ngo 1.26.0\n"), 0o644); err != nil {
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

func TestGoWork_resolveForstRuntimeLink_invalidReplaceFallsThrough(t *testing.T) {
	root := t.TempDir()
	forstGomod := filepath.Join(root, ".forst-gomod")
	if err := os.MkdirAll(forstGomod, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(forstGomod, "go.mod"), []byte("module app\n\nreplace forst => ./missing-forst-runtime\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	compilerDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(compilerDir, "cmd", "forst"), 0o755); err != nil {
		t.Fatal(err)
	}
	restore := goload.SetForstCompilerModuleRootHookForTest(func() string { return compilerDir })
	defer restore()

	link, err := ResolveForstRuntimeLink(root)
	if err != nil {
		t.Fatal(err)
	}
	if link.ReplaceDir != filepath.Clean(compilerDir) {
		t.Fatalf("ReplaceDir = %q want compiler fallback %q", link.ReplaceDir, compilerDir)
	}
}

func TestGoWork_planForRun_usesWorkspaceForGoNativeModule(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/app\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	compilerDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(compilerDir, "cmd", "forst"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(compilerDir, "go.mod"), []byte("module forst\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	restore := goload.SetForstCompilerModuleRootHookForTest(func() string { return compilerDir })
	defer restore()

	session := filepath.Join(dir, ".forst", "run", "dev")
	plan, err := PlanForRun(dir, session, true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Mode != LinkWorkspace {
		t.Fatalf("want LinkWorkspace for Go-native module, got %v", plan.Mode)
	}
	wantWork := filepath.Join(dir, ".forst", "go.work")
	if plan.Workspace != wantWork {
		t.Fatalf("Workspace = %q want %q", plan.Workspace, wantWork)
	}
}

func TestGoWork_planForRun_ftconfigBoundaryUsesReplace(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module multi_package_dev\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"server":{"embedded":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	compilerDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(compilerDir, "cmd", "forst"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(compilerDir, "go.mod"), []byte("module forst\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	restore := goload.SetForstCompilerModuleRootHookForTest(func() string { return compilerDir })
	defer restore()

	session := filepath.Join(dir, ".forst", "run", "dev")
	plan, err := PlanForRun(dir, session, true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Mode != LinkReplace {
		t.Fatalf("want LinkReplace for ftconfig boundary, got %v", plan.Mode)
	}
}

func TestPlanForRun_testSessionUsesReplace(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/app\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	compilerDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(compilerDir, "cmd", "forst"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(compilerDir, "go.mod"), []byte("module forst\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	restore := goload.SetForstCompilerModuleRootHookForTest(func() string { return compilerDir })
	defer restore()

	session := filepath.Join(dir, ".forst", "gen", "test", "s1", "mod")
	plan, err := PlanForRun(dir, session, true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Mode != LinkReplace {
		t.Fatalf("want LinkReplace for forst test session, got %v", plan.Mode)
	}
}

func TestGoWork_planForRun_forstGomodShimUsesReplace(t *testing.T) {
	dir := t.TempDir()
	forstGomod := filepath.Join(dir, ".forst-gomod")
	if err := os.MkdirAll(forstGomod, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(forstGomod, "go.mod"), []byte("module example.com/app/forst\n\ngo 1.26.0\nrequire forst v0.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	compilerDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(compilerDir, "cmd", "forst"), 0o755); err != nil {
		t.Fatal(err)
	}
	restore := goload.SetForstCompilerModuleRootHookForTest(func() string { return compilerDir })
	defer restore()

	plan, err := PlanForRun(dir, filepath.Join(dir, ".forst", "run", "s1"), true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Mode != LinkReplace {
		t.Fatalf("want LinkReplace for .forst-gomod shim, got %v", plan.Mode)
	}
}

func TestWriteGoWork_listsSandboxAndCompiler(t *testing.T) {
	dir := t.TempDir()
	workPath := filepath.Join(dir, ".forst", "go.work")
	sandbox := filepath.Join(dir, ".forst", "run", "dev")
	compiler := filepath.Join(dir, "compiler", "forst")
	if err := os.MkdirAll(sandbox, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(compiler, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteGoWork(workPath, []string{sandbox, compiler}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(workPath)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "use (") || !strings.Contains(s, "run/dev") || !strings.Contains(s, "../compiler/forst") {
		t.Fatalf("unexpected go.work:\n%s", s)
	}
}

func TestWriteRunGoMod_workspaceMode_omitsReplaceForst(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "go.mod")
	link := ForstRuntimeLink{ReplaceDir: filepath.Join(dir, "forst")}
	if err := WriteRunGoMod(path, link, "", "", true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if strings.Contains(s, "replace forst") {
		t.Fatalf("workspace mode must not write replace forst:\n%s", s)
	}
	if !strings.Contains(s, "require forst v0.0.0") {
		t.Fatalf("missing require forst:\n%s", s)
	}
}

func TestGoModReplacePath_relativeWhenSameTree(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "compiler", "forst")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	modDir := filepath.Join(dir, "sandbox")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := goModReplacePath(modDir, target)
	if err != nil {
		t.Fatal(err)
	}
	if got != "../compiler/forst" {
		t.Fatalf("goModReplacePath = %q want ../compiler/forst", got)
	}
}

func TestGoModReplacePath_absoluteWhenPrivateSandboxToUsersTarget(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	compilerDir, err := filepath.Abs(filepath.Join(filepath.Dir(file), "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	compilerDir = resolveCanonicalPath(compilerDir)
	if strings.HasPrefix(compilerDir, "/private/") {
		t.Skip("compiler module under /private; need /Users vs /private split")
	}
	got, err := goModReplacePath("/private/var/folders/xx/T/sandbox", compilerDir)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.ToSlash(compilerDir)
	if got != want {
		t.Fatalf("goModReplacePath = %q want absolute %q", got, want)
	}
	if strings.HasPrefix(got, "../") {
		t.Fatalf("expected absolute path, got relative %q", got)
	}
}

func TestWriteRunGoMod_absoluteReplaceWhenCrossTree(t *testing.T) {
	sandbox := resolveCanonicalPath(t.TempDir())
	if !strings.HasPrefix(sandbox, "/private/") {
		t.Skip("sandbox not under /private (macOS /var symlink); skipping cross-tree test")
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	compilerDir, err := filepath.Abs(filepath.Join(filepath.Dir(file), "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	compilerDir = resolveCanonicalPath(compilerDir)
	if strings.HasPrefix(compilerDir, "/private/") {
		t.Skip("compiler module also under /private; need /Users vs /private split")
	}

	path := filepath.Join(sandbox, "go.mod")
	link := ForstRuntimeLink{ReplaceDir: compilerDir}
	if err := WriteRunGoMod(path, link, "", "", false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if strings.Contains(s, "replace forst => ../") {
		t.Fatalf("expected absolute replace forst across /private vs /Users, got:\n%s", s)
	}
	want := "replace forst => " + filepath.ToSlash(compilerDir)
	if !strings.Contains(s, want) {
		t.Fatalf("missing absolute replace forst => %s:\n%s", compilerDir, s)
	}
	if _, err := exec.LookPath("go"); err == nil {
		cmd := exec.Command("go", "mod", "tidy")
		cmd.Dir = sandbox
		cmd.Env = append(os.Environ(), "GOWORK=off")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go mod tidy in cross-tree sandbox: %v\n%s", err, out)
		}
	}
}
