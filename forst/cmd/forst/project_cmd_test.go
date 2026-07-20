package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func writeMinimalProjectRoot(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module projecttest\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.ft"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestRunProjectInfo_printsProjectLayout(t *testing.T) {
	dir := t.TempDir()
	writeMinimalProjectRoot(t, dir)
	log := logrus.New()
	log.SetOutput(io.Discard)

	out := captureStdout(t, func() {
		if code := runProjectInfo([]string{"-root", dir}, log); code != 0 {
			t.Fatalf("runProjectInfo code = %d", code)
		}
	})
	for _, want := range []string{"boundary:", "moduleRoot:", "modulePath:", "devProfile:", "packages:", "runnableExports:"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q:\n%s", want, out)
		}
	}
}

func TestRunClean_dryRunLeavesDotForst(t *testing.T) {
	dir := t.TempDir()
	writeMinimalProjectRoot(t, dir)
	dotForst := filepath.Join(dir, ".forst")
	if err := os.MkdirAll(filepath.Join(dotForst, "run"), 0o755); err != nil {
		t.Fatal(err)
	}

	log := logrus.New()
	var logs bytes.Buffer
	log.SetOutput(&logs)

	if code := runClean([]string{"-root", dir, "-dry-run", "-verbose"}, log); code != 0 {
		t.Fatalf("runClean code = %d logs=%s", code, logs.String())
	}
	if !strings.Contains(logs.String(), "would remove") {
		t.Fatalf("expected dry-run log, got %q", logs.String())
	}
	if _, err := os.Stat(dotForst); err != nil {
		t.Fatalf(".forst removed during dry-run: %v", err)
	}
}

func TestRunClean_removesDotForst(t *testing.T) {
	dir := t.TempDir()
	writeMinimalProjectRoot(t, dir)
	dotForst := filepath.Join(dir, ".forst")
	if err := os.MkdirAll(filepath.Join(dotForst, "run"), 0o755); err != nil {
		t.Fatal(err)
	}

	log := logrus.New()
	log.SetOutput(io.Discard)
	if code := runClean([]string{"-root", dir}, log); code != 0 {
		t.Fatalf("runClean code = %d", code)
	}
	if _, err := os.Stat(dotForst); !os.IsNotExist(err) {
		t.Fatalf(".forst still exists: %v", err)
	}
}

func TestRunClean_missingDotForst_isNoop(t *testing.T) {
	dir := t.TempDir()
	writeMinimalProjectRoot(t, dir)
	log := logrus.New()
	var logs bytes.Buffer
	log.SetOutput(&logs)

	if code := runClean([]string{"-root", dir}, log); code != 0 {
		t.Fatalf("runClean code = %d", code)
	}
	if !strings.Contains(logs.String(), "nothing to remove") {
		t.Fatalf("logs = %q", logs.String())
	}
}
