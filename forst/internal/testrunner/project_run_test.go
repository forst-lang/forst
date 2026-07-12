package testrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWithProject_crossPkg_noZFilesInSourceTree(t *testing.T) {
	stubGoTestSuccess(t)
	dir := t.TempDir()
	writeCrossPkgSandboxFixture(t, dir)
	code, err := Run(Options{
		ModuleRoot: dir,
		Paths:      []string{"./api"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if code != ExitSuccess {
		t.Fatalf("expected success, got %d", code)
	}
	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if strings.HasPrefix(base, "z_forst_gen") {
			return &zFileError{path: path}
		}
		return nil
	})
	if err != nil {
		if ze, ok := err.(*zFileError); ok {
			t.Fatalf("forst test must not write %s", ze.path)
		}
		t.Fatal(err)
	}
}

func writeCrossPkgSandboxFixture(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module cross_sandbox\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	authDir := filepath.Join(dir, "auth")
	apiDir := filepath.Join(dir, "api")
	for _, d := range []string{authDir, apiDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(authDir, "log.ft"), []byte(`package auth

type Logger = {Info(msg String)}

func LogEvent(id String) {
	use logger: Logger
	logger.Info("expire " + id)
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "handle.ft"), []byte(`package api

import "cross_sandbox/auth"

type Logger = {Info(msg String)}

func HandleRequest(id String) {
	auth.LogEvent(id)
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "handle_test.ft"), []byte(`package api

import "testing"

type NopLogger = {}

func (NopLogger) Info(msg String) {}

func TestHandleRequest_propagatesLoggerFromAuth(t *testing.T) {
	with {Logger: &NopLogger{}} {
		HandleRequest("tok-1")
	}
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
}

type zFileError struct{ path string }

func (e *zFileError) Error() string { return e.path }
