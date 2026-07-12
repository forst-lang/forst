package compiler

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"forst/internal/goload"
	transformer_go "forst/internal/transformer/go"

	"github.com/sirupsen/logrus"
)

func testCompilerLogger() *logrus.Logger {
	log := logrus.New()
	log.SetOutput(io.Discard)
	log.SetLevel(logrus.ErrorLevel)
	return log
}

// TestProgramCompilation smoke-checks compile; full examples/in matrix lives in cmd/forst TestExamples.
func TestProgramCompilation(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "valid basic program",
			filePath: "../../../examples/in/basic.ft",
			wantErr:  false,
		},
		{
			name:     "non-existent file",
			filePath: "nonexistent.ft",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := New(Args{
				Command:  "run",
				FilePath: tt.filePath,
				LogLevel: "error",
			}, testCompilerLogger())

			code, err := c.CompileFile()
			if (err != nil) != tt.wantErr {
				t.Errorf("Program.compileFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && code == nil {
				t.Error("Program.compileFile() returned nil code for successful compilation")
			}
		})
	}
}

func TestTempOutputFile(t *testing.T) {
	testCode := "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}"

	outputPath, err := CreateTempOutputFile(testCode)
	if err != nil {
		t.Fatalf("CreateTempOutputFile() error = %v", err)
	}

	defer func() {
		err := os.RemoveAll(filepath.Dir(outputPath))
		if err != nil {
			t.Errorf("Failed to remove temporary output file: %v", err)
		}
	}()

	// Verify the file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("CreateTempOutputFile() did not create the output file")
	}

	// Verify the content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != testCode {
		t.Errorf("CreateTempOutputFile() content = %v, want %v", string(content), testCode)
	}
}

func TestCompileFile_exportStructFields_emitsJSONMarshalableShape(t *testing.T) {
	dir := t.TempDir()
	typesPath := filepath.Join(dir, "types.ft")
	if err := os.WriteFile(typesPath, []byte(`package demo

type Wire = {
	msg: String,
	count: Int,
}

func Identity(x Wire): Wire {
	return x
}
`), 0644); err != nil {
		t.Fatal(err)
	}

	c := New(Args{
		Command:            "build",
		FilePath:           typesPath,
		PackageRoot:        dir,
		LogLevel:           "error",
		ExportStructFields: true,
	}, nil)

	code, err := c.CompileFile()
	if err != nil {
		t.Fatalf("CompileFile: %v", err)
	}
	if code == nil {
		t.Fatal("nil code")
	}
	out := *code
	if !strings.Contains(out, "Msg") || !strings.Contains(out, "Count") {
		t.Fatalf("expected exported field names Msg and Count, got: %.400s", out)
	}
	if !strings.Contains(out, "`json:\"msg\"`") || !strings.Contains(out, "`json:\"count\"`") {
		t.Fatalf("expected json struct tags for Forst field names, got: %.400s", out)
	}
}

func TestCompileFile_unionErrorNarrowing_example(t *testing.T) {
	t.Parallel()
	c := New(Args{
		Command:  "build",
		FilePath: "../../../examples/in/union_error_narrowing.ft",
		LogLevel: "error",
	}, nil)
	code, err := c.CompileFile()
	if err != nil {
		t.Fatalf("CompileFile: %v", err)
	}
	if code == nil {
		t.Fatal("nil code")
	}
	out := *code
	for _, want := range []string{
		"type ErrKind interface",
		"isErrKind()",
		"type ParseError struct",
		"func onlyParseError(p ParseError)",
		"func demo()",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("generated Go missing %q\n%.1200s", want, out)
		}
	}
}

func TestCompileFile_nominalErrorExample_emitsNamedPayloadStruct(t *testing.T) {
	c := New(Args{
		Command:  "build",
		FilePath: "../../../examples/in/nominal_error.ft",
		LogLevel: "error",
	}, nil)
	code, err := c.CompileFile()
	if err != nil {
		t.Fatalf("CompileFile: %v", err)
	}
	if code == nil {
		t.Fatal("nil code")
	}
	out := *code
	for _, want := range []string{
		"type NotPositive struct",
		"message string",
		"func Test() error",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("generated Go missing %q; output (truncated):\n%.800s", want, out)
		}
	}
}

func TestCompileFile_packageRootMergesSamePackage(t *testing.T) {
	dir := t.TempDir()
	typesPath := filepath.Join(dir, "types.ft")
	apiPath := filepath.Join(dir, "api.ft")
	if err := os.WriteFile(typesPath, []byte(`package demo

type Greeting = String
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(apiPath, []byte(`package demo

func Main(): Greeting {
	return "ok"
}
`), 0644); err != nil {
		t.Fatal(err)
	}

	c := New(Args{
		Command:     "build",
		FilePath:    apiPath,
		PackageRoot: dir,
		LogLevel:    "error",
	}, nil)

	code, err := c.CompileFile()
	if err != nil {
		t.Fatalf("CompileFile: %v", err)
	}
	if code == nil || len(*code) < 10 {
		t.Fatalf("expected generated Go code")
	}
}

func TestRunBoundaryRoot_prefersPackageRoot(t *testing.T) {
	dir := t.TempDir()
	got := RunBoundaryRoot(Args{
		PackageRoot: "/project/root",
		FilePath:    filepath.Join(dir, "main.ft"),
	})
	if got != "/project/root" {
		t.Fatalf("got %q want package root", got)
	}
}

func TestRunBoundaryRoot_fallsBackToEntryDir(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.ft")
	got := RunBoundaryRoot(Args{FilePath: entry})
	want, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFormatRunProgramError_wrapsExitError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		runExit    int
		wantSubstr []string
		wantSame   bool
	}{
		{
			name:       "exit code 1",
			runExit:    1,
			wantSubstr: []string{"generated program exited with code 1", "tsx"},
		},
		{
			name:       "exit code 2 generic hint",
			runExit:    2,
			wantSubstr: []string{"generated program exited with code 2", "see stderr above for details"},
		},
		{
			name:     "non-exit error passthrough",
			err:      errors.New("boom"),
			wantSame: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var err error
			if tc.err != nil {
				err = tc.err
			} else {
				cmd := exec.Command("sh", "-c", "exit "+strconv.Itoa(tc.runExit))
				err = cmd.Run()
				if err == nil {
					t.Fatal("expected exit error")
				}
			}

			wrapped := formatRunProgramError(err, "")
			if tc.wantSame {
				if wrapped != err {
					t.Fatalf("got %v want same %v", wrapped, err)
				}
				return
			}
			for _, substr := range tc.wantSubstr {
				if !strings.Contains(wrapped.Error(), substr) {
					t.Fatalf("got %q, want substring %q", wrapped, substr)
				}
			}
		})
	}
}

func TestRunGoProgram(t *testing.T) {
	// Create a temporary test program
	testCode := "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}"
	outputPath, err := CreateTempOutputFile(testCode)
	if err != nil {
		t.Fatalf("Failed to create test program: %v", err)
	}

	defer func() {
		err := os.RemoveAll(filepath.Dir(outputPath))
		if err != nil {
			t.Errorf("Failed to remove temporary output file: %v", err)
		}
	}()

	// Test running the program
	err = RunGoProgram(outputPath, "")
	if err != nil {
		t.Errorf("RunGoProgram() error = %v", err)
	}
}

func TestLoadInputNodesForCompile_singleFilePath(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "single.ft")
	src := `package main

func Main() {
	return
}
`
	if err := os.WriteFile(filePath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	c := New(Args{
		Command:  "build",
		FilePath: filePath,
		LogLevel: "error",
	}, nil)
	nodes, err := c.loadInputNodesForCompile()
	if err != nil {
		t.Fatalf("loadInputNodesForCompile: %v", err)
	}
	if len(nodes) == 0 {
		t.Fatal("expected parsed nodes for single-file path")
	}
}

func TestLoadInputNodesForCompile_packageRootEntryOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outsideDir := t.TempDir()
	entry := filepath.Join(outsideDir, "entry.ft")
	if err := os.WriteFile(entry, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := New(Args{
		Command:     "build",
		FilePath:    entry,
		PackageRoot: root,
		LogLevel:    "error",
	}, nil)
	_, err := c.loadInputNodesForCompile()
	if err == nil {
		t.Fatal("expected error when entry file is outside package root")
	}
	if !strings.Contains(err.Error(), "is not under -root") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompileFile_outputPathWriteError(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "ok.ft")
	src := `package main

func Main() {
	return
}
`
	if err := os.WriteFile(filePath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(dir, "missing-dir", "out.go")
	c := New(Args{
		Command:    "build",
		FilePath:   filePath,
		OutputPath: outputPath,
		LogLevel:   "error",
	}, nil)
	_, err := c.CompileFile()
	if err == nil {
		t.Fatal("expected output path write error")
	}
	if !strings.Contains(err.Error(), "error writing output file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompileFile_traceLogLevel_printsGeneratedCodeToStdout(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "trace.ft")
	src := `package main

func Main() {
	return
}
`
	if err := os.WriteFile(filePath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	originalStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = writePipe
	t.Cleanup(func() { os.Stdout = originalStdout })

	c := New(Args{
		Command:  "build",
		FilePath: filePath,
		LogLevel: "trace",
	}, nil)
	_, err = c.CompileFile()
	if err != nil {
		t.Fatalf("CompileFile: %v", err)
	}

	if err := writePipe.Close(); err != nil {
		t.Fatalf("close stdout write pipe: %v", err)
	}
	stdoutBytes, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	stdout := string(stdoutBytes)
	if !strings.Contains(stdout, "package main") {
		t.Fatalf("expected generated code on stdout, got: %s", stdout)
	}
}

// TestCompileFile_map_catalog_golden keeps examples/out/map_catalog.go aligned with codegen for
// examples/in/map_catalog.ft (also exercised by cmd/forst TestExamples).
// Regenerate: UPDATE_MAP_CATALOG_GOLDEN=1 go test ./internal/compiler -run TestCompileFile_map_catalog_golden -count=1
func TestCompileFile_deterministic_basicExample(t *testing.T) {
	path := filepath.Join("..", "..", "..", "examples", "in", "basic.ft")
	const runs = 2
	compiled := make([]*string, runs)
	for i := range runs {
		var err error
		c := New(Args{Command: "build", FilePath: path, LogLevel: "error"}, testCompilerLogger())
		compiled[i], err = c.CompileFile()
		if err != nil {
			t.Fatalf("CompileFile (run %d): %v", i+1, err)
		}
	}
	ref := *compiled[0]
	for i := range runs {
		if *compiled[i] != ref {
			t.Fatalf("non-deterministic output: run %d != first", i)
		}
	}
}

func TestCompileFile_map_catalog_golden(t *testing.T) {
	c := New(Args{
		Command:  "run",
		FilePath: "../../../examples/in/map_catalog.ft",
		LogLevel: "error",
	}, nil)
	code, err := c.CompileFile()
	if err != nil {
		t.Fatalf("CompileFile: %v", err)
	}
	actual := *code
	goldenPath := filepath.Join("..", "..", "..", "examples", "out", "map_catalog.go")
	if os.Getenv("UPDATE_MAP_CATALOG_GOLDEN") == "1" || os.Getenv("UPDATE_EXAMPLES_GOLDENS") == "1" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, []byte(actual), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", goldenPath)
		return
	}
	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v (set UPDATE_MAP_CATALOG_GOLDEN=1 to create)", goldenPath, err)
	}
	verifyMapCatalogGolden(t, string(expected), actual, goldenPath)
}

func verifyMapCatalogGolden(t *testing.T, expected, actual, goldenPath string) {
	t.Helper()
	markers := []string{
		"package main",
		"func main()",
		`var errMissingMapKey = errors.New("missing map key")`,
		`missing map key`,
		`v, ok :=`,
		`!ok`,
		`errMissingMapKey`,
		`fmt`,
		`fmt.Println`,
		`catalog[sku]`,
	}
	for _, m := range markers {
		if !strings.Contains(actual, m) {
			t.Errorf("generated Go missing %q (golden %s)", m, goldenPath)
		}
	}
	if len(expected) > 0 && len(actual) < len(expected)/2 {
		t.Errorf("output much shorter than golden (%d vs %d bytes)", len(actual), len(expected))
	}
}

func forstCompilerModuleFromTest(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	moduleRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	if !goload.IsForstCompilerModule(moduleRoot) {
		t.Skip("forst compiler module not found from test location")
	}
	return moduleRoot
}

func TestCreateTempOutputFiles_companionUsesForstModuleRoot(t *testing.T) {
	forstMod := forstCompilerModuleFromTest(t)
	t.Setenv(goload.EnvForstGOModRoot, forstMod)
	outside := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(outside); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	outputPath, err := CreateTempOutputFiles(
		"package main\n\nfunc main() {}\n",
		"package main\n",
		"",
		nil,
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateTempOutputFiles: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(outputPath)) })

	wantPrefix := filepath.Join(forstMod, ".forst", "run")
	if !strings.HasPrefix(outputPath, wantPrefix) {
		t.Fatalf("outputPath %q not under %q", outputPath, wantPrefix)
	}
}

func TestGoModuleRootForRun_companionFiles(t *testing.T) {
	forstMod := forstCompilerModuleFromTest(t)
	tempDir, err := os.MkdirTemp("", "forst-companion-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	invokePath := filepath.Join(tempDir, transformer_go.ForstInvokeServerFileName()+".go")
	if err := os.WriteFile(invokePath, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(mainPath, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := goModuleRootForRun(mainPath); got != forstMod {
		t.Fatalf("goModuleRootForRun() = %q want %q", got, forstMod)
	}
}

func TestRunGoProgram_companionImportsCompile(t *testing.T) {
	forstCompilerModuleFromTest(t)
	err := BuildGoProgram(
		"package main\n\nimport _ \"forst/nodert\"\n\nfunc main() {}\n",
		"package main\n",
		"",
	)
	if err != nil {
		t.Fatalf("BuildGoProgram: %v", err)
	}
}

func TestCreateTempOutputFiles_emptyForstLink_returnsError(t *testing.T) {
	root := t.TempDir()
	restore := goload.SetForstCompilerModuleRootHookForTest(func() string { return "" })
	defer restore()

	boundary := filepath.Join(root, "app")
	if err := os.MkdirAll(boundary, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := CreateTempOutputFiles(
		"package main\n\nfunc main() {}\n",
		"package main\n",
		"",
		nil,
		nil,
		boundary,
	)
	if err == nil {
		t.Fatal("expected error when forst runtime link is missing")
	}
	if !strings.Contains(err.Error(), ".forst-gomod/go.mod") {
		t.Fatalf("unexpected error: %v", err)
	}
}
