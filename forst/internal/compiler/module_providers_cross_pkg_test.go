package compiler

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"forst/internal/goload"
)

func TestPackageRootModuleProvidersPath_nodeInteropUsesFreshChecker(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	syncRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "examples", "in", "rfc", "node-interop", "sync"))
	modRoot := goload.FindModuleRoot(syncRoot)
	if moduleRootHasGoMod(modRoot) {
		t.Fatalf("node-interop sync should not resolve to a go.mod module root, got modRoot=%q", modRoot)
	}
	c := New(Args{PackageRoot: syncRoot, FilePath: filepath.Join(syncRoot, "main.ft")}, nil)
	if c.packageRootHasStandaloneModuleProviders() {
		t.Fatal("expected fresh checker path for -root under sync")
	}
}

func TestCompileWithNodeRuntime_syncPackageRoot(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "examples", "in", "rfc", "node-interop", "sync"))
	c := New(Args{
		Command:            "build",
		FilePath:           filepath.Join(root, "main.ft"),
		PackageRoot:        root,
		ExportStructFields: true,
		LogLevel:           "error",
	}, silentCompilerTestLogger())
	if c.packageRootHasStandaloneModuleProviders() {
		t.Fatal("sync -root should not use standalone module providers pass")
	}
	main, runtime, _, err := c.CompileWithNodeRuntime()
	if err != nil {
		t.Fatalf("CompileWithNodeRuntime: %v", err)
	}
	if main == "" || !strings.Contains(main, "forst_node_callsync_") {
		t.Fatalf("missing bridge wrapper in main:\n%s", main)
	}
	if runtime == "" || !strings.Contains(runtime, "nodert.CallSync") {
		t.Fatalf("missing nodert wiring in runtime:\n%s", runtime)
	}
}

func TestCompileFile_packageRoot_crossPkgWithEmittedGoStub(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module cross_stub\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	authDir := filepath.Join(dir, "auth")
	apiDir := filepath.Join(dir, "api")
	for _, d := range []string{authDir, apiDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	const emittedGo = `package auth

type Providers_stub struct {
	Logger any
}

func LogEvent(providers Providers_stub, id string) {}
`
	if err := os.WriteFile(filepath.Join(authDir, "z_forst_gen.go"), []byte(emittedGo), 0o644); err != nil {
		t.Fatal(err)
	}
	const authFt = `package auth

type Logger = { Info(msg String) }

func LogEvent(id String) {
	use logger: Logger
	logger.Info(id)
}
`
	const apiFt = `package api

import "cross_stub/auth"

func HandleRequest(id String) {
	auth.LogEvent(id)
}
`
	apiPath := filepath.Join(apiDir, "handle.ft")
	if err := os.WriteFile(filepath.Join(authDir, "log.ft"), []byte(authFt), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(apiPath, []byte(apiFt), 0o644); err != nil {
		t.Fatal(err)
	}

	c := New(Args{
		Command:     "build",
		FilePath:    apiPath,
		PackageRoot: apiDir,
		LogLevel:    "error",
	}, silentCompilerTestLogger())
	code, err := c.CompileFile()
	if err != nil {
		t.Fatalf("CompileFile with -root should use Forst sibling auth.LogEvent, not Go stub arity: %v", err)
	}
	if code == nil {
		t.Fatal("expected non-nil code")
	}
	out := *code
	for _, sub := range []string{
		`func HandleRequest(providers`,
		`auth.LogEvent(`,
		`auth.Providers_`,
		`Logger:`,
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("missing %q in generated api output:\n%s", sub, out)
		}
	}
}
