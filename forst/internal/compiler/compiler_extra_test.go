package compiler

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestParseArgs_delegatesToOsArgs(t *testing.T) {
	old := os.Args
	t.Cleanup(func() { os.Args = old })
	os.Args = []string{"forst", "build", "-o", "out.go", "main.ft"}
	log := logrus.New()
	log.SetOutput(io.Discard)
	args := ParseArgs(log)
	if args.Command != "build" || args.FilePath != "main.ft" || args.OutputPath != "out.go" {
		t.Fatalf("args = %+v", args)
	}
}

func TestParseArgsFrom_invalidRootPath(t *testing.T) {
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	// Abs rarely fails on Unix; use an overlong path where supported.
	longRoot := strings.Repeat("a", 4096)
	args := ParseArgsFrom([]string{"forst", "run", "-root", longRoot, "x.ft"}, log)
	if args.PackageRoot != "" {
		// Platform accepted the path; still exercised ParseArgsFrom run branch.
		if args.Command != "run" || args.FilePath != "x.ft" {
			t.Fatalf("args = %+v", args)
		}
		return
	}
	if args.Command != "" {
		t.Fatalf("expected empty args, got %+v", args)
	}
	if !strings.Contains(buf.String(), "invalid -root") {
		t.Fatalf("log = %q", buf.String())
	}
}

func TestGoWorkspaceDirForCheck_usesPackageRootOrEntryDir(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	moduleRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	basic := filepath.Join(moduleRoot, "..", "examples", "in", "basic.ft")

	withRoot := New(Args{PackageRoot: filepath.Join(moduleRoot, "..", "examples", "in")}, nil)
	if got := withRoot.goWorkspaceDirForCheck(); got == "" {
		t.Fatal("expected module root from -root")
	}

	withFile := New(Args{FilePath: basic}, nil)
	if got := withFile.goWorkspaceDirForCheck(); got == "" {
		t.Fatal("expected module root from entry dir")
	}
}

func TestIsCompilerWorkspaceModule_detectsForstRepo(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	moduleRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	if !isCompilerWorkspaceModule(moduleRoot) {
		t.Fatal("expected forst module to be detected as compiler workspace")
	}
	if isCompilerWorkspaceModule(t.TempDir()) {
		t.Fatal("temp dir should not be compiler workspace")
	}
}

func TestTypecheckForCompile_moduleProvidersError(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "host.ft")
	if err := os.WriteFile(ft, []byte(`package host

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
	c := New(Args{Command: "build", FilePath: ft, LogLevel: "error"}, silentCompilerTestLogger())
	nodes, err := c.lexParseEntryFile()
	if err != nil {
		t.Fatal(err)
	}
	tc, _, err := c.typecheckForCompile(nodes)
	if err == nil {
		t.Fatal("expected module providers error")
	}
	if tc != nil {
		t.Fatal("expected nil checker on module error")
	}
}

func TestCompileFile_typecheckErrorPrintsScope(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "bad.ft")
	if err := os.WriteFile(ft, []byte(`package main

func Broken(): String {
	return 1
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.ErrorLevel)
	c := New(Args{Command: "build", FilePath: ft, LogLevel: "error"}, log)
	if _, err := c.CompileFile(); err == nil {
		t.Fatal("expected typecheck error")
	}
	if !strings.Contains(buf.String(), "Encountered error checking types") {
		t.Fatalf("log = %q", buf.String())
	}
}

func TestCompileFile_tracePrintsGeneratedCode(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "ok.ft")
	if err := os.WriteFile(ft, []byte(`package main

func main() {
	println("hi")
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	c := New(Args{Command: "build", FilePath: ft, LogLevel: "trace"}, silentCompilerTestLogger())
	code, err := c.CompileFile()
	if err != nil || code == nil || !strings.Contains(*code, "func main") {
		t.Fatalf("CompileFile: err=%v code=%v", err, code)
	}
}

func TestCompileFile_outputWriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses chmod")
	}
	dir := t.TempDir()
	ft := filepath.Join(dir, "ok.ft")
	outDir := filepath.Join(dir, "out")
	if err := os.Mkdir(outDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(outDir, 0o755) })
	if err := os.WriteFile(ft, []byte(`package main

func main() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(outDir, "gen.go")
	c := New(Args{Command: "build", FilePath: ft, OutputPath: outPath, LogLevel: "error"}, silentCompilerTestLogger())
	if _, err := c.CompileFile(); err == nil || !strings.Contains(err.Error(), "writing output file") {
		t.Fatalf("err = %v", err)
	}
}

func TestLexParseEntryFile_missingFile(t *testing.T) {
	c := New(Args{FilePath: filepath.Join(t.TempDir(), "missing.ft"), LogLevel: "error"}, silentCompilerTestLogger())
	if _, err := c.lexParseEntryFile(); err == nil || !strings.Contains(err.Error(), "reading file") {
		t.Fatalf("err = %v", err)
	}
}

func TestCollectSamePackageFtPaths_invalidRoot(t *testing.T) {
	_, err := collectSamePackageFtPaths(silentCompilerTestLogger(), string([]byte{0}), filepath.Join(t.TempDir(), "x.ft"))
	if err == nil {
		t.Fatal("expected open root error")
	}
}

func TestLoadMergedPackageAST_invalidAbsPaths(t *testing.T) {
	c := New(Args{
		PackageRoot: string([]byte{'/', 0, 'r'}),
		FilePath:    string([]byte{'/', 0, 'e'}),
	}, silentCompilerTestLogger())
	if _, err := c.loadMergedPackageAST(); err == nil {
		t.Fatal("expected abs error")
	}
}

func TestWatchFile_recompilesOnWrite(t *testing.T) {
	origRun := runGoProgramForWatch
	t.Cleanup(func() { runGoProgramForWatch = origRun })
	var runs atomic.Int32
	runGoProgramForWatch = func(string) error {
		runs.Add(1)
		return nil
	}

	dir := t.TempDir()
	ft := filepath.Join(dir, "w.ft")
	out := filepath.Join(dir, "out.go")
	src := `package main

func main() {
	println("v1")
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, []byte("// placeholder"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := New(Args{FilePath: ft, OutputPath: out, LogLevel: "error"}, silentCompilerTestLogger())
	go func() { _ = c.WatchFile() }()

	deadline := time.Now().Add(3 * time.Second)
	for runs.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if runs.Load() < 1 {
		t.Fatal("expected initial compileAndRunOnce")
	}

	src2 := strings.Replace(src, "v1", "v2", 1)
	if err := os.WriteFile(ft, []byte(src2), 0o644); err != nil {
		t.Fatal(err)
	}
	for runs.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if runs.Load() < 2 {
		t.Fatalf("expected recompile on write, runs=%d", runs.Load())
	}
}
