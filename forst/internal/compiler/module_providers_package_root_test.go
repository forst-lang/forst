package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
