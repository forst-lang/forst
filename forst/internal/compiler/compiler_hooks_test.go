package compiler

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/forstpkg"
	transformer_go "forst/internal/transformer/go"

	goast "go/ast"

	"github.com/sirupsen/logrus"
)

func TestCompileFile_transformError(t *testing.T) {
	orig := transformForstFileToGoCompile
	t.Cleanup(func() { transformForstFileToGoCompile = orig })
	transformForstFileToGoCompile = func(*transformer_go.Transformer, []ast.Node) (*goast.File, error) {
		return nil, errors.New("transform failed")
	}

	dir := t.TempDir()
	ft := filepath.Join(dir, "ok.ft")
	if err := os.WriteFile(ft, []byte(`package main

func main() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	c := New(Args{Command: "build", FilePath: ft, LogLevel: "error"}, silentCompilerTestLogger())
	if _, err := c.CompileFile(); err == nil || !strings.Contains(err.Error(), "transform failed") {
		t.Fatalf("err = %v", err)
	}
}

func TestCompileFile_generateError(t *testing.T) {
	origGen := generateGoCodeCompile
	t.Cleanup(func() { generateGoCodeCompile = origGen })
	generateGoCodeCompile = func(*goast.File) (string, error) {
		return "", errors.New("generate failed")
	}

	dir := t.TempDir()
	ft := filepath.Join(dir, "ok.ft")
	if err := os.WriteFile(ft, []byte(`package main

func main() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	c := New(Args{Command: "build", FilePath: ft, LogLevel: "error"}, silentCompilerTestLogger())
	if _, err := c.CompileFile(); err == nil || !strings.Contains(err.Error(), "generate failed") {
		t.Fatalf("err = %v", err)
	}
}

func TestCreateTempOutputFile_mkdirError(t *testing.T) {
	orig := mkdirTemp
	t.Cleanup(func() { mkdirTemp = orig })
	mkdirTemp = func(string, string) (string, error) {
		return "", errors.New("mkdir failed")
	}
	if _, err := CreateTempOutputFile("package main\n"); err == nil {
		t.Fatal("expected mkdir error")
	}
}

func TestCreateTempOutputFile_writeError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses chmod")
	}
	orig := mkdirTemp
	t.Cleanup(func() { mkdirTemp = orig })
	readOnly := filepath.Join(t.TempDir(), "ro")
	if err := os.Mkdir(readOnly, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnly, 0o755) })
	mkdirTemp = func(string, string) (string, error) {
		return readOnly, nil
	}
	if _, err := CreateTempOutputFile("package main\n"); err == nil || !strings.Contains(err.Error(), "failed to write temp file") {
		t.Fatalf("err = %v", err)
	}
}

func TestIsCompilerWorkspaceModule_emptyRoot(t *testing.T) {
	if isCompilerWorkspaceModule("") {
		t.Fatal("empty root should be false")
	}
}

func TestTypecheckForCompileEntry_loadError(t *testing.T) {
	c := New(Args{
		Command:  "build",
		FilePath: filepath.Join(t.TempDir(), "missing.ft"),
		LogLevel: "error",
	}, silentCompilerTestLogger())
	if _, _, err := c.TypecheckForCompileEntry(); err == nil {
		t.Fatal("expected load error")
	}
}

func TestTypecheckForCompile_usesPerPackageWhenPresent(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	moduleRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	basic := filepath.Join(moduleRoot, "..", "examples", "in", "basic.ft")
	if _, err := os.Stat(basic); err != nil {
		t.Skip("examples/in/basic.ft not available")
	}
	c := New(Args{Command: "build", FilePath: basic, LogLevel: "error"}, silentCompilerTestLogger())
	nodes, err := c.lexParseEntryFile()
	if err != nil {
		t.Fatal(err)
	}
	tc, modResult, err := c.typecheckForCompile(nodes)
	if err != nil || tc == nil || modResult == nil {
		t.Fatalf("typecheck: err=%v tc=%v modResult=%v", err, tc, modResult)
	}
	pkg := forstpkg.PackageNameOrDefault(forstpkg.PackageNameFromNodes(nodes))
	if modResult.PerPackage[pkg] == nil {
		t.Fatal("expected module PerPackage entry")
	}
	if c.typecheckUsesFreshEntryChecker(filepath.Dir(basic)) {
		if tc == modResult.PerPackage[pkg] {
			t.Fatal("examples/in entry compile must re-typecheck the entry file AST")
		}
	} else if modResult.PerPackage[pkg] != tc {
		t.Fatal("expected typechecker from module PerPackage map")
	}
}

func TestTypecheckForCompile_usesFallbackWhenNoPerPackage(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "solo.ft")
	if err := os.WriteFile(ft, []byte(`package solo

func ok(): Int { return 1 }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	c := New(Args{Command: "build", FilePath: ft, LogLevel: "error"}, silentCompilerTestLogger())
	nodes, err := c.lexParseEntryFile()
	if err != nil {
		t.Fatal(err)
	}
	tc, modResult, err := c.typecheckForCompile(nodes)
	if err != nil || tc == nil {
		t.Fatalf("typecheck: err=%v tc=%v modResult=%v", err, tc, modResult)
	}
}

func TestLexParseEntryFile_debugPrintsTokensAndAST(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "ok.ft")
	if err := os.WriteFile(ft, []byte(`package main

func main() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.DebugLevel)
	c := New(Args{Command: "build", FilePath: ft, LogLevel: "debug"}, log)
	if _, err := c.lexParseEntryFile(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Tokens") || !strings.Contains(out, "Forst AST") {
		t.Fatalf("log = %q", out)
	}
}

func TestCompileFile_debugPrintsTypeAndGoAST(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "ok.ft")
	if err := os.WriteFile(ft, []byte(`package main

func main() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.DebugLevel)
	c := New(Args{Command: "build", FilePath: ft, LogLevel: "debug"}, log)
	if _, err := c.CompileFile(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Type Check Results") || !strings.Contains(out, "Go AST") {
		t.Fatalf("log = %q", out)
	}
}

func TestTypecheckForCompile_fallbackCheckTypesErrorReturnsChecker(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "other.ft"), []byte(`package other

func Ok(): Int { return 1 }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	soloPath := filepath.Join(dir, "solo.ft")
	c := New(Args{Command: "build", FilePath: soloPath, LogLevel: "error"}, silentCompilerTestLogger())
	if err := os.WriteFile(soloPath, []byte(`package solo

func bad(): String { return 1 }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	nodes, err := c.lexParseEntryFile()
	if err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(soloPath)

	tc, _, err := c.typecheckForCompile(nodes)
	if err == nil || tc == nil {
		t.Fatalf("expected fallback typecheck error with checker, err=%v tc=%v", err, tc)
	}
}
