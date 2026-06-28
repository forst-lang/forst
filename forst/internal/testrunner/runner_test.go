package testrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	code, err := emitPackageGo(dir, pkgs[0], nil, log)
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

func TestRun_providersWithWiringPasses(t *testing.T) {
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

func TestParseCLIArgs_splitsPathsAndGoTestFlags(t *testing.T) {
	paths, goArgs := ParseCLIArgs([]string{"-v", "./auth", "--", "-count=1"})
	if len(paths) != 1 || paths[0] != "./auth" {
		t.Fatalf("paths = %v", paths)
	}
	if len(goArgs) != 2 || goArgs[0] != "-count=1" || goArgs[1] != "-v" {
		t.Fatalf("goArgs = %v", goArgs)
	}
}
