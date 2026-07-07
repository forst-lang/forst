package testrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/modulecheck"
	"forst/internal/testmod"

	"github.com/sirupsen/logrus"
)

func writeProvidersTestFixture(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("testmod")), 0o644); err != nil {
		t.Fatal(err)
	}
	authDir := filepath.Join(dir, "auth")
	if err := os.MkdirAll(authDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(authDir, "auth.ft"), []byte(`package auth

type Logger = { info(msg String) }

type NopLogger = {}

func (NopLogger) info(msg String) {}

func expireToken() {
	use logger: Logger
	logger.info("ok")
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(authDir, "auth_test.ft"), []byte(`package auth

import "testing"

func TestExpireWithWiring(t *testing.T) {
	with { Logger: &NopLogger {} } {
		expireToken()
	}
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverPackages_findsTestPackage(t *testing.T) {
	dir := t.TempDir()
	writeProvidersTestFixture(t, dir)
	pkgs, err := DiscoverPackages(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 || pkgs[0].RelPath != "auth" {
		t.Fatalf("pkgs = %+v", pkgs)
	}
	if len(pkgs[0].TestPaths) != 1 || !strings.HasSuffix(pkgs[0].TestPaths[0], "auth_test.ft") {
		t.Fatalf("test paths = %v", pkgs[0].TestPaths)
	}
	if len(pkgs[0].FtPaths) < 2 {
		t.Fatalf("expected merged ft paths, got %v", pkgs[0].FtPaths)
	}
}

func TestEmit_mergedPackageTestFunctionSignature(t *testing.T) {
	dir := t.TempDir()
	writeProvidersTestFixture(t, dir)
	pkgs, err := DiscoverPackages(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetOutput(os.Stderr)
	log.SetLevel(logrus.PanicLevel)
	code, err := emitPackageGo(dir, pkgs[0], nil, EmitOptions{}, log)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(code, "func TestExpireWithWiring(providers ") {
		t.Fatalf("test wiring root should not get providers param:\n%s", code)
	}
	if !strings.Contains(code, "func TestExpireWithWiring(t *testing.T)") {
		t.Fatalf("expected Go test signature, got:\n%s", code)
	}
}

func TestWriteGeneratedTestAndRun_requiresGoMod(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetOutput(os.Stderr)
	log.SetLevel(logrus.PanicLevel)
	code, err := writeGeneratedTestAndRun(PackageUnderTest{
		Dir:     pkgDir,
		RelPath: "pkg",
	}, "package pkg\n", nil, log)
	if err == nil || code == ExitSuccess {
		t.Fatalf("expected go.mod error, code=%d err=%v", code, err)
	}
	if !strings.Contains(err.Error(), "no go.mod") {
		t.Fatalf("err = %v", err)
	}
}

func TestRun_e2e_realGoTest(t *testing.T) {
	dir := t.TempDir()
	writeProvidersTestFixture(t, dir)
	log := logrus.New()
	log.SetOutput(os.Stderr)
	log.SetLevel(logrus.PanicLevel)
	code, err := Run(Options{ModuleRoot: dir, Log: log})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("exit code = %d, want %d", code, ExitSuccess)
	}
	if _, err := os.Stat(filepath.Join(dir, "auth", generatedTestGoName)); !os.IsNotExist(err) {
		t.Fatal("expected generated test file removed after run")
	}
}

func TestRun_orchestration_stubbedGoTest(t *testing.T) {
	stubGoTestSuccess(t)
	dir := t.TempDir()
	writeProvidersTestFixture(t, dir)
	code, err := Run(Options{ModuleRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d", code)
	}
	if _, err := os.Stat(filepath.Join(dir, "auth", generatedTestGoName)); !os.IsNotExist(err) {
		t.Fatal("expected generated test file removed after stubbed go test")
	}
}

func TestRelPath(t *testing.T) {
	t.Parallel()
	moduleRoot := filepath.Clean("/proj")
	if got := relPath(moduleRoot, filepath.Join(moduleRoot, "auth")); got != "auth" {
		t.Fatalf("relPath = %q", got)
	}
	if got := relPath(moduleRoot, moduleRoot); got != "." {
		t.Fatalf("same dir relPath = %q", got)
	}
}

func TestIndexOf(t *testing.T) {
	t.Parallel()
	if indexOf([]string{"a", "--", "b"}, "--") != 1 {
		t.Fatal("expected index 1")
	}
	if indexOf([]string{"a"}, "--") != -1 {
		t.Fatal("expected -1")
	}
}

func TestEmitDependencyPackages_writesLibraryGo(t *testing.T) {
	dir := t.TempDir()
	writeProvidersTestFixture(t, dir)
	libDir := filepath.Join(dir, "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "lib.ft"), []byte(`package lib

func Helper(): Int { return 1 }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetOutput(os.Stderr)
	log.SetLevel(logrus.PanicLevel)
	modResult, err := modulecheck.CheckModuleProviders(log, modulecheck.Options{ModuleRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	testDirs := map[string]struct{}{filepath.Join(dir, "auth"): {}}
	if _, err := emitDependencyPackages(dir, modResult, testDirs, EmitOptions{}, log); err != nil {
		t.Fatal(err)
	}
	genPath := filepath.Join(libDir, "z_forst_gen.go")
	data, err := os.ReadFile(genPath)
	if err != nil {
		t.Fatalf("expected generated lib Go: %v", err)
	}
	if !strings.Contains(string(data), "Helper") {
		t.Fatalf("generated code missing Helper:\n%s", data)
	}
}

func TestParseCLIArgs_splitsPathsAndGoTestFlags(t *testing.T) {
	t.Parallel()
	paths, goArgs := ParseCLIArgs([]string{"-v", "./auth", "--", "-count=1"})
	if len(paths) != 1 || paths[0] != "./auth" {
		t.Fatalf("paths = %v", paths)
	}
	if len(goArgs) != 2 || goArgs[0] != "-count=1" || goArgs[1] != "-v" {
		t.Fatalf("goArgs = %v", goArgs)
	}
}

func writeLibClientTestFixture(t *testing.T, dir string) (libpkgDir, clientDir string) {
	t.Helper()
	const mod = "generic_lib_test"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent(mod)), 0o644); err != nil {
		t.Fatal(err)
	}
	libpkgDir = filepath.Join(dir, "libpkg")
	clientDir = filepath.Join(dir, "client")
	for _, d := range []string{libpkgDir, clientDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(libpkgDir, "lib.ft"), []byte(`package libpkg

type Widget = {
	Id: String,
}

func Ping() {
	println("ping")
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(libpkgDir, "lib_test.ft"), []byte(`package libpkg

import "testing"

func TestPing(t *testing.T) {
	Ping()
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(clientDir, "client.ft"), []byte(`package client

import "generic_lib_test/libpkg"

func Run() {
	libpkg.Ping()
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(clientDir, "client_test.ft"), []byte(`package client

import "testing"

func TestRun(t *testing.T) {
	Run()
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	return libpkgDir, clientDir
}

func TestEmitPackageGo_testOnlyOmitsPackageTypeDefs(t *testing.T) {
	dir := t.TempDir()
	libpkgDir, _ := writeLibClientTestFixture(t, dir)
	log := logrus.New()
	log.SetOutput(os.Stderr)
	log.SetLevel(logrus.PanicLevel)

	modResult, err := modulecheck.CheckModuleProviders(log, modulecheck.Options{ModuleRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	pkgs, err := DiscoverPackages(dir, []string{"libpkg"})
	if err != nil {
		t.Fatal(err)
	}
	pkg := pkgs[0]

	libCode, err := emitPackageGo(dir, pkg, modResult, EmitOptions{}, log)
	if err != nil {
		t.Fatal(err)
	}
	genPath := filepath.Join(libpkgDir, "z_forst_gen.go")
	if err := os.WriteFile(genPath, []byte(libCode), 0o644); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(libCode, "type Widget") {
		t.Fatalf("library emit should define Widget:\n%s", libCode)
	}

	testCode, err := emitPackageGo(dir, pkg, modResult, EmitOptions{TestOnly: true}, log)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(testCode, "type Widget") {
		t.Fatalf("test-only emit must not redeclare package types:\n%s", testCode)
	}
	if !strings.Contains(testCode, "func TestPing(t *testing.T)") {
		t.Fatalf("test-only emit missing test function:\n%s", testCode)
	}
}

func writeIfScopeTestFixture(t *testing.T, dir string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("ifscope_test")), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgDir := filepath.Join(dir, "demo")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "lib.ft"), []byte(`package demo

func buildBody(label String): String {
	return label
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "lib_test.ft"), []byte(`package demo

import "testing"

func TestBodyNonEmpty(t *testing.T) {
	body := buildBody("x")
	if body == "" {
		t.Fatal("empty body")
	}
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	return pkgDir
}

func TestEmitPackageGo_testOnlyIfScopeUsesMergedParseNodes(t *testing.T) {
	dir := t.TempDir()
	pkgDir := writeIfScopeTestFixture(t, dir)
	log := logrus.New()
	log.SetOutput(os.Stderr)
	log.SetLevel(logrus.PanicLevel)

	modResult, err := modulecheck.CheckModuleProviders(log, modulecheck.Options{ModuleRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	pkgs, err := DiscoverPackages(dir, []string{"demo"})
	if err != nil {
		t.Fatal(err)
	}
	pkg := pkgs[0]

	libCode, err := emitPackageGo(dir, pkg, modResult, EmitOptions{}, log)
	if err != nil {
		t.Fatalf("emit lib: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "z_forst_gen.go"), []byte(libCode), 0o644); err != nil {
		t.Fatal(err)
	}

	testCode, err := emitPackageGo(dir, pkg, modResult, EmitOptions{TestOnly: true}, log)
	if err != nil {
		t.Fatalf("emit test-only: %v", err)
	}
	if !strings.Contains(testCode, "func TestBodyNonEmpty(t *testing.T)") {
		t.Fatalf("test-only emit missing test function:\n%s", testCode)
	}
}

func TestRun_dependencyEmitThenSelfTest_succeeds(t *testing.T) {
	stubGoTestSuccess(t)
	dir := t.TempDir()
	libpkgDir, _ := writeLibClientTestFixture(t, dir)
	log := logrus.New()
	log.SetOutput(os.Stderr)
	log.SetLevel(logrus.PanicLevel)

	code, err := Run(Options{ModuleRoot: dir, Paths: []string{"client"}, Log: log})
	if err != nil {
		t.Fatalf("Run client: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("Run client exit = %d, want %d", code, ExitSuccess)
	}
	libGen, err := os.ReadFile(filepath.Join(libpkgDir, "z_forst_gen.go"))
	if err != nil {
		t.Fatalf("expected libpkg z_forst_gen.go after client test: %v", err)
	}
	if !strings.Contains(string(libGen), "type Widget") {
		t.Fatalf("library shim missing Widget:\n%s", libGen)
	}

	code, err = Run(Options{ModuleRoot: dir, Paths: []string{"libpkg"}, Log: log})
	if err != nil {
		t.Fatalf("Run libpkg: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("Run libpkg exit = %d, want %d", code, ExitSuccess)
	}
}
