package main

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/testmod"
	"forst/internal/testrunner"

	"github.com/sirupsen/logrus"
)

func testCmdLogger() *logrus.Logger {
	log := logrus.New()
	log.SetOutput(os.Stderr)
	log.SetLevel(logrus.PanicLevel)
	return log
}

func writeForstTestFixture(t *testing.T, dir string) {
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

func TestRunTestCommand_success(t *testing.T) {
	dir := t.TempDir()
	writeForstTestFixture(t, dir)
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	code := runTestCommand([]string{"./auth"}, testCmdLogger())
	if code != testrunner.ExitSuccess.Int() {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestRunTestCommand_noTestsFound(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("testmod")), 0o644); err != nil {
		t.Fatal(err)
	}
	libDir := filepath.Join(dir, "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "lib.ft"), []byte(`package lib

func F(): Int { return 1 }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	code := runTestCommand(nil, testCmdLogger())
	if code != testrunner.ExitError.Int() {
		t.Fatalf("exit code = %d, want 2 when no *_test.ft packages", code)
	}
}

func TestRunTestCommand_invalidModuleRoot(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	// Path outside any go.mod — goload.ModuleRootWithGoMod fails for explicit dir arg.
	code := runTestCommand([]string{"./missing"}, testCmdLogger())
	if code != 2 {
		t.Fatalf("exit code = %d, want 2 for setup error", code)
	}
}

func TestRunTestCommand_resolvesModuleRootFromSubdir(t *testing.T) {
	dir := t.TempDir()
	writeForstTestFixture(t, dir)
	authDir := filepath.Join(dir, "auth")
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(authDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	code := runTestCommand([]string{"."}, testCmdLogger())
	if code != testrunner.ExitSuccess.Int() {
		t.Fatalf("exit code = %d from subdir", code)
	}
}

func TestRunTestCommand_passesGoTestArgs(t *testing.T) {
	dir := t.TempDir()
	writeForstTestFixture(t, dir)
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	// -run=NonExistent skips the only test → go test exits 0 with no tests run.
	code := runTestCommand([]string{"--", "-run=NoSuchTest"}, testCmdLogger())
	if code != testrunner.ExitSuccess.Int() {
		t.Fatalf("exit code = %d", code)
	}
}

func TestRunTestCommand_typecheckFailure(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("testmod")), 0o644); err != nil {
		t.Fatal(err)
	}
	badDir := filepath.Join(dir, "bad")
	if err := os.MkdirAll(badDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badDir, "bad.ft"), []byte(`package bad

func broken(): Int { return "not int" }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badDir, "bad_test.ft"), []byte(`package bad

import "testing"

func TestBroken(t *testing.T) {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	code := runTestCommand([]string{"./bad"}, testCmdLogger())
	if code != testrunner.ExitFailure.Int() {
		t.Fatalf("exit code = %d, want 1 for typecheck failure", code)
	}
}

func TestRunTestCommand_relDotNormalizesPaths(t *testing.T) {
	dir := t.TempDir()
	writeForstTestFixture(t, dir)
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	var captured []string
	orig := runTestCommandRunner
	runTestCommandRunner = func(opts testrunner.Options) (testrunner.ExitCode, error) {
		captured = opts.Paths
		return testrunner.ExitSuccess, nil
	}
	t.Cleanup(func() { runTestCommandRunner = orig })

	code := runTestCommand([]string{"."}, testCmdLogger())
	if code != testrunner.ExitSuccess.Int() {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if len(captured) != 0 {
		t.Fatalf("paths = %v, want empty (walk module)", captured)
	}
}

func TestRunTestCommand_moduleRootPathFindsNestedTests(t *testing.T) {
	dir := t.TempDir()
	writeForstTestFixture(t, dir)

	code := runTestCommand([]string{dir}, testCmdLogger())
	if code != testrunner.ExitSuccess.Int() {
		t.Fatalf("exit code = %d, want 0 for module root with nested *_test.ft", code)
	}
}

func TestRunTestCommand_getwdError(t *testing.T) {
	orig := runTestCommandGetwd
	runTestCommandGetwd = func() (string, error) { return "", os.ErrInvalid }
	t.Cleanup(func() { runTestCommandGetwd = orig })
	if code := runTestCommand(nil, testCmdLogger()); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
}

func TestRunTestCommand_pathAbsError(t *testing.T) {
	dir := t.TempDir()
	writeForstTestFixture(t, dir)
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	orig := runTestCommandPathAbs
	runTestCommandPathAbs = func(string) (string, error) { return "", os.ErrInvalid }
	t.Cleanup(func() { runTestCommandPathAbs = orig })

	if code := runTestCommand([]string{"./auth"}, testCmdLogger()); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
}

func TestRunTestCommand_pathRelError(t *testing.T) {
	dir := t.TempDir()
	writeForstTestFixture(t, dir)
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	orig := runTestCommandPathRel
	runTestCommandPathRel = func(string, string) (string, error) { return "", os.ErrInvalid }
	t.Cleanup(func() { runTestCommandPathRel = orig })

	if code := runTestCommand([]string{"./auth"}, testCmdLogger()); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
}

func TestRunTestCommand_dirWithoutGoMod(t *testing.T) {
	dir := t.TempDir()
	noMod := filepath.Join(dir, "plain")
	if err := os.MkdirAll(noMod, 0o755); err != nil {
		t.Fatal(err)
	}
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	if code := runTestCommand([]string{"plain"}, testCmdLogger()); code != 2 {
		t.Fatalf("exit code = %d, want 2 for directory without go.mod", code)
	}
}

func TestRunTestCommand_runnerErrorWithSuccessCode(t *testing.T) {
	orig := runTestCommandRunner
	runTestCommandRunner = func(testrunner.Options) (testrunner.ExitCode, error) {
		return testrunner.ExitSuccess, os.ErrInvalid
	}
	t.Cleanup(func() { runTestCommandRunner = orig })
	if code := runTestCommand(nil, testCmdLogger()); code != testrunner.ExitError.Int() {
		t.Fatalf("exit code = %d, want 2", code)
	}
}

func TestRunTestCommand_runnerErrorWithFailureCode(t *testing.T) {
	orig := runTestCommandRunner
	runTestCommandRunner = func(testrunner.Options) (testrunner.ExitCode, error) {
		return testrunner.ExitFailure, os.ErrInvalid
	}
	t.Cleanup(func() { runTestCommandRunner = orig })
	if code := runTestCommand(nil, testCmdLogger()); code != testrunner.ExitFailure.Int() {
		t.Fatalf("exit code = %d, want 1", code)
	}
}

func TestRunTestCommand_multiplePackagePaths(t *testing.T) {
	dir := t.TempDir()
	writeForstTestFixture(t, dir)
	domainDir := filepath.Join(dir, "domain")
	if err := os.MkdirAll(domainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(domainDir, "domain.ft"), []byte(`package domain

func Edge(): String { return "ok" }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(domainDir, "domain_test.ft"), []byte(`package domain

import "testing"

func TestEdge(t *testing.T) {
	if Edge() != "ok" {
		t.Fatal("edge")
	}
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	var captured []string
	orig := runTestCommandRunner
	runTestCommandRunner = func(opts testrunner.Options) (testrunner.ExitCode, error) {
		captured = append([]string(nil), opts.Paths...)
		return testrunner.ExitSuccess, nil
	}
	t.Cleanup(func() { runTestCommandRunner = orig })

	code := runTestCommand([]string{"./auth", "./domain"}, testCmdLogger())
	if code != testrunner.ExitSuccess.Int() {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if len(captured) != 2 {
		t.Fatalf("paths = %v, want auth and domain", captured)
	}
	want := map[string]bool{"auth": true, "domain": true}
	for _, p := range captured {
		if !want[p] {
			t.Fatalf("unexpected path %q in %v", p, captured)
		}
	}
}

func TestRunTestCommand_skipsEmptyPathEntry(t *testing.T) {
	dir := t.TempDir()
	writeForstTestFixture(t, dir)
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	code := runTestCommand([]string{"", "./auth"}, testCmdLogger())
	if code != testrunner.ExitSuccess.Int() {
		t.Fatalf("exit code = %d", code)
	}
}
