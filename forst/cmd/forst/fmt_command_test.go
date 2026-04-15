package main

import (
	"os"
	"strings"
	"testing"

	"forst/internal/printer"
)

func TestRunFmtCommand_WritesFormattedContent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initial := "package main  \nfunc main() {\n}\n"
	path := writeFmtTestFile(t, dir, "a.ft", initial, 0o600)

	_, err := runFmt(t, path)
	if err != nil {
		t.Fatal(err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) == initial {
		t.Fatal("expected file to be rewritten")
	}
	want := printer.FormatDocument(initial, path, 8, false, testFmtLogger())
	if string(after) != want {
		t.Fatalf("got %q want %q", after, want)
	}
}

func TestRunFmtCommand_PreservesOriginalPermissionsOnWrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initial := "package main  \nfunc main() {\n}\n"
	path := writeFmtTestFile(t, dir, "a.ft", initial, 0o600)

	beforeInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	_, err = runFmt(t, path)
	if err != nil {
		t.Fatal(err)
	}

	afterInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if afterInfo.Mode().Perm() != beforeInfo.Mode().Perm() {
		t.Fatalf("expected mode %o after write, got %o", beforeInfo.Mode().Perm(), afterInfo.Mode().Perm())
	}
}

func TestRunFmtCommand_ListDoesNotWrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initial := "package main  \nfunc main() {\n}\n"
	path := writeFmtTestFile(t, dir, "a.ft", initial, 0o600)

	out, err := runFmt(t, "-l", path)
	if err != nil {
		t.Fatal(err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != initial {
		t.Fatal("-l must not write")
	}
	if got := strings.TrimSpace(out); got != path {
		t.Fatalf("list output = %q want %q", got, path)
	}
}

func TestRunFmtCommand_DryRunPrintsWriteLines(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initial := "package main  \nfunc main() {\n}\n"
	path := writeFmtTestFile(t, dir, "a.ft", initial, 0o600)

	out, err := runFmt(t, "-n", path)
	if err != nil {
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
	if strings.TrimSpace(out) != wantLine {
		t.Fatalf("got %q want %q", out, wantLine)
	}
}

func TestRunFmtCommand_ListPrecedenceOverDryRun(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initial := "package main  \nfunc main() {\n}\n"
	path := writeFmtTestFile(t, dir, "a.ft", initial, 0o600)

	out, err := runFmt(t, "-l", "-n", path)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out); got != path {
		t.Fatalf("expected -l output precedence, got %q want %q", got, path)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != initial {
		t.Fatal("combined -l -n must not write")
	}
}

func TestRunFmtCommand_DefaultPathCurrentDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initial := "package main  \nfunc main() {\n}\n"
	writeFmtTestFile(t, dir, "a.ft", initial, 0o600)

	previousWorkingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWorkingDir)
	})

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	out, err := runFmt(t, "-l")
	if err != nil {
		t.Fatal(err)
	}

	if got := strings.TrimSpace(out); got != "a.ft" {
		t.Fatalf("expected default path to use cwd and list %q, got %q", "a.ft", got)
	}
}
