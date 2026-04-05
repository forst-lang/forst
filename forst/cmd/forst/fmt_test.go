package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/printer"

	"github.com/sirupsen/logrus"
)

func TestRunFmtCommand_WritesFormattedContent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "a.ft")
	initial := "package main  \nfunc main() {\n}\n"
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	var buf bytes.Buffer
	if err := runFmtCommand([]string{path}, log, &buf); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) == initial {
		t.Fatal("expected file to be rewritten")
	}
	want := printer.FormatDocument(initial, path, 8, false, log)
	if string(after) != want {
		t.Fatalf("got %q want %q", after, want)
	}
}

func TestRunFmtCommand_ListDoesNotWrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "a.ft")
	initial := "package main  \nfunc main() {\n}\n"
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	var buf bytes.Buffer
	if err := runFmtCommand([]string{"-l", path}, log, &buf); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != initial {
		t.Fatal("-l must not write")
	}
	out := strings.TrimSpace(buf.String())
	if out != path {
		t.Fatalf("list output = %q want %q", out, path)
	}
}

func TestRunFmtCommand_DryRunPrintsWriteLines(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "a.ft")
	initial := "package main  \nfunc main() {\n}\n"
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	var buf bytes.Buffer
	if err := runFmtCommand([]string{"-n", path}, log, &buf); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != initial {
		t.Fatal("-n must not write")
	}
	wantLine := "write " + path
	if strings.TrimSpace(buf.String()) != wantLine {
		t.Fatalf("got %q want %q", buf.String(), wantLine)
	}
}

func TestCollectFtPaths_SkipsNonFt(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	goFile := filepath.Join(dir, "x.go")
	if err := os.WriteFile(goFile, []byte("package x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := collectFtPaths([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}
