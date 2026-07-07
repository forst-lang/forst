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

func testLog(t *testing.T) *logrus.Logger {
	t.Helper()
	log := logrus.New()
	log.SetOutput(os.Stderr)
	log.SetLevel(logrus.PanicLevel)
	return log
}

func TestRelPath_returnsDirWhenRelFails(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(string(filepath.Separator), "abs", "pkg")
	if got := relPath("", dir); got != dir {
		t.Fatalf("relPath = %q, want %q", got, dir)
	}
}

func TestRun_moduleProvidersFailure(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("badmod")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "host.ft"), []byte(`package host

import "testing"

type Logger = { info(msg String) }
type NopLogger = {}

func (NopLogger) info(msg String) {}

func TestHost(t *testing.T) {
	with { BadKey: &NopLogger {} } {
	}
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "host_test.ft"), []byte(`package host

import "testing"

func TestHostCase(t *testing.T) {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	code, err := Run(Options{ModuleRoot: dir, Log: testLog(t)})
	if err == nil || code != ExitFailure {
		t.Fatalf("code=%d err=%v, want ExitFailure with error", code, err)
	}
	if !strings.Contains(err.Error(), "module providers") {
		t.Fatalf("err = %v", err)
	}
}

func TestRun_onePackageFailsOthersStillRun(t *testing.T) {
	stubGoTestFailImport(t, "fail")
	dir := t.TempDir()
	writeProvidersTestFixture(t, dir)
	failDir := filepath.Join(dir, "fail")
	if err := os.MkdirAll(failDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(failDir, "fail.ft"), []byte(`package fail

func failCheck(name String) {
	ensure name is Min(1)
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(failDir, "fail_test.ft"), []byte(`package fail

import "testing"

func TestFail(t *testing.T) {
	failCheck("")
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	code, err := Run(Options{ModuleRoot: dir, Log: testLog(t)})
	if err != nil {
		t.Fatal(err)
	}
	if code != ExitFailure {
		t.Fatalf("code = %d, want ExitFailure", code)
	}
}

func TestEmitPackageGo_parseError(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "bad")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pkg := PackageUnderTest{
		Dir:     pkgDir,
		RelPath: "bad",
		FtPaths: []string{filepath.Join(pkgDir, "bad_test.ft")},
	}
	if err := os.WriteFile(pkg.FtPaths[0], []byte("package bad\nfunc {"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := emitPackageGo(dir, pkg, nil, EmitOptions{}, testLog(t))
	if err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("err = %v", err)
	}
}

func TestEmitPackageGo_testOnlyNoTestPaths(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	lib := filepath.Join(pkgDir, "pkg.ft")
	testFt := filepath.Join(pkgDir, "pkg_test.ft")
	if err := os.WriteFile(lib, []byte(`package pkg

func ok(): Int { return 1 }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testFt, []byte(`package pkg

import "testing"

func TestOk(t *testing.T) {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg := PackageUnderTest{
		Dir:     pkgDir,
		RelPath: "pkg",
		FtPaths: []string{lib, testFt},
	}
	_, err := emitPackageGo(dir, pkg, nil, EmitOptions{TestOnly: true}, testLog(t))
	if err == nil || !strings.Contains(err.Error(), "no test paths") {
		t.Fatalf("err = %v", err)
	}
}

func TestEmitPackageGo_testOnlyMergeTransformError(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	lib := filepath.Join(pkgDir, "pkg.ft")
	testFt := filepath.Join(pkgDir, "pkg_test.ft")
	if err := os.WriteFile(lib, []byte(`package pkg

func ok(): Int { return 1 }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testFt, []byte(`package pkg

import "testing"

func TestOk(t *testing.T) {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg := PackageUnderTest{
		Dir:       pkgDir,
		RelPath:   "pkg",
		FtPaths:   []string{lib},
		TestPaths: []string{testFt},
	}
	_, err := emitPackageGo(dir, pkg, nil, EmitOptions{TestOnly: true}, testLog(t))
	if err == nil || !strings.Contains(err.Error(), "merge transform nodes") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunPackageTests_emitFailure(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "bad")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pkg := PackageUnderTest{
		Dir:     pkgDir,
		RelPath: "bad",
		FtPaths: []string{filepath.Join(pkgDir, "bad_test.ft")},
	}
	if err := os.WriteFile(pkg.FtPaths[0], []byte("package bad\n<<<"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, err := runPackageTests(dir, pkg, nil, EmitOptions{}, nil, testLog(t))
	if err == nil || code != ExitFailure {
		t.Fatalf("code=%d err=%v", code, err)
	}
}

func TestWriteGeneratedTestAndRun_writeError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses chmod 000")
	}
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("pkgmod")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(pkgDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(pkgDir, 0o755) })
	code, err := writeGeneratedTestAndRun(PackageUnderTest{
		Dir:     pkgDir,
		RelPath: "pkg",
	}, "package pkg\n", nil, testLog(t))
	if err == nil || code != ExitFailure {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if !strings.Contains(err.Error(), "write generated test") {
		t.Fatalf("err = %v", err)
	}
}

func TestWriteGeneratedTestAndRun_testFailureExitCode(t *testing.T) {
	stubGoTestExit(t, 1)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("failmod")), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgDir := filepath.Join(dir, "auth")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	failCode := `package auth

import "testing"

func TestFail(t *testing.T) {
	t.Fatal("fail")
}
`
	exit, err := writeGeneratedTestAndRun(PackageUnderTest{
		Dir:     pkgDir,
		RelPath: "auth",
	}, failCode, nil, testLog(t))
	if err != nil {
		t.Fatal(err)
	}
	if exit == ExitSuccess {
		t.Fatalf("expected non-zero exit, got %d", exit)
	}
}

func TestEmitDependencyPackages_skipsTestDirsAndEmptyLib(t *testing.T) {
	dir := t.TempDir()
	writeProvidersTestFixture(t, dir)
	libDir := filepath.Join(dir, "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Only test sources — libPaths stays empty after filtering.
	if err := os.WriteFile(filepath.Join(libDir, "lib_test.ft"), []byte(`package lib

import "testing"

func TestLib(t *testing.T) {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	modResult, err := modulecheck.CheckModuleProviders(testLog(t), modulecheck.Options{ModuleRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	testDirs := map[string]struct{}{filepath.Join(dir, "auth"): {}}
	if _, err := emitDependencyPackages(dir, modResult, testDirs, EmitOptions{}, testLog(t)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(libDir, "z_forst_gen.go")); !os.IsNotExist(err) {
		t.Fatal("expected no generated lib Go for test-only package")
	}
}

func TestEmitDependencyPackages_emitError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("emiterr")), 0o644); err != nil {
		t.Fatal(err)
	}
	libDir := filepath.Join(dir, "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "lib.ft"), []byte("package lib\nfunc {"), 0o644); err != nil {
		t.Fatal(err)
	}
	modResult := &modulecheck.ModuleResult{
		ForstPkgToFiles: map[string][]string{
			"lib": {filepath.Join(libDir, "lib.ft")},
		},
	}
	_, err := emitDependencyPackages(dir, modResult, nil, EmitOptions{}, testLog(t))
	if err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("err = %v", err)
	}
}

func TestEmitDependencyPackages_writeError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses chmod 000")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("writerr")), 0o644); err != nil {
		t.Fatal(err)
	}
	libDir := filepath.Join(dir, "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "lib.ft"), []byte(`package lib

func Helper(): Int { return 1 }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	modResult := &modulecheck.ModuleResult{
		ForstPkgToFiles: map[string][]string{
			"lib": {filepath.Join(libDir, "lib.ft")},
		},
	}
	if err := os.Chmod(libDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(libDir, 0o755) })
	_, err := emitDependencyPackages(dir, modResult, nil, EmitOptions{}, testLog(t))
	if err == nil || !strings.Contains(err.Error(), "write generated") {
		t.Fatalf("err = %v", err)
	}
}

func TestRun_defaultModuleRootAndGoTestLookupError(t *testing.T) {
	dir := t.TempDir()
	writeProvidersTestFixture(t, dir)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	t.Setenv("PATH", filepath.Join(t.TempDir(), "empty-path"))
	code, err := Run(Options{Log: testLog(t)})
	if err == nil || code != ExitError {
		t.Fatalf("code=%d err=%v", code, err)
	}
}

func TestRun_discoverPackagesError(t *testing.T) {
	dir := t.TempDir()
	code, err := Run(Options{ModuleRoot: dir, Paths: []string{"missing"}, Log: testLog(t)})
	if err == nil || code != ExitError {
		t.Fatalf("code=%d err=%v", code, err)
	}
}

func TestEmitPackageGo_typecheckFailureInFallback(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "bad")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	lib := filepath.Join(pkgDir, "bad.ft")
	ft := filepath.Join(pkgDir, "bad_test.ft")
	if err := os.WriteFile(lib, []byte(`package bad

func broken(x Int): String {
	return x
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ft, []byte(`package bad

import "testing"

func TestBroken(t *testing.T) {
	_ = broken(1)
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg := PackageUnderTest{Dir: pkgDir, RelPath: "bad", FtPaths: []string{lib, ft}}
	_, err := emitPackageGo(dir, pkg, nil, EmitOptions{}, testLog(t))
	if err == nil || !strings.Contains(err.Error(), "typecheck") {
		t.Fatalf("err = %v", err)
	}
}

func TestEmitPackageGo_usesModulePerPackageChecker(t *testing.T) {
	dir := t.TempDir()
	writeProvidersTestFixture(t, dir)
	modResult, err := modulecheck.CheckModuleProviders(testLog(t), modulecheck.Options{ModuleRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	pkgDir := filepath.Join(dir, "auth")
	pkg := PackageUnderTest{
		Dir:     pkgDir,
		RelPath: "auth",
		FtPaths: []string{
			filepath.Join(pkgDir, "auth.ft"),
			filepath.Join(pkgDir, "auth_test.ft"),
		},
	}
	code, err := emitPackageGo(dir, pkg, modResult, EmitOptions{}, testLog(t))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(code, "TestExpireWithWiring") {
		t.Fatalf("missing test in output")
	}
}

func TestEmitDependencyPackages_skipsEmptyFileList(t *testing.T) {
	dir := t.TempDir()
	modResult := &modulecheck.ModuleResult{
		ForstPkgToFiles: map[string][]string{"empty": {}},
	}
	if _, err := emitDependencyPackages(dir, modResult, nil, EmitOptions{}, testLog(t)); err != nil {
		t.Fatal(err)
	}
}

func TestWriteGeneratedTestAndRun_successRemovesGeneratedFile(t *testing.T) {
	stubGoTestSuccess(t)
	dir := t.TempDir()
	writeProvidersTestFixture(t, dir)
	pkgDir := filepath.Join(dir, "auth")
	code, err := emitPackageGo(dir, PackageUnderTest{
		Dir:     pkgDir,
		RelPath: "auth",
		FtPaths: []string{
			filepath.Join(pkgDir, "auth.ft"),
			filepath.Join(pkgDir, "auth_test.ft"),
		},
	}, nil, EmitOptions{}, testLog(t))
	if err != nil {
		t.Fatal(err)
	}
	exit, err := writeGeneratedTestAndRun(PackageUnderTest{
		Dir:     pkgDir,
		RelPath: "auth",
	}, code, nil, testLog(t))
	if err != nil || exit != ExitSuccess {
		t.Fatalf("exit=%d err=%v", exit, err)
	}
	if _, err := os.Stat(filepath.Join(pkgDir, generatedTestGoName)); !os.IsNotExist(err) {
		t.Fatal("expected generated file removed after successful go test")
	}
}

func TestWriteGeneratedTestAndRun_nestedImportPath(t *testing.T) {
	stubGoTestSuccess(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("nested")), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgDir := filepath.Join(dir, "nested", "pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	code := `package pkg

import "testing"

func TestOk(t *testing.T) {}
`
	exit, err := writeGeneratedTestAndRun(PackageUnderTest{
		Dir:     pkgDir,
		RelPath: "nested/pkg",
	}, code, nil, testLog(t))
	if err != nil || exit != ExitSuccess {
		t.Fatalf("exit=%d err=%v", exit, err)
	}
}

func TestRun_emitDependencyPackagesFailure(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses chmod 000")
	}
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
	if err := os.Chmod(libDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(libDir, 0o755) })
	code, err := Run(Options{ModuleRoot: dir, Log: testLog(t)})
	if err == nil || code != ExitFailure {
		t.Fatalf("code=%d err=%v", code, err)
	}
}
