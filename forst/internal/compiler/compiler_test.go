package compiler

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
			name:     "valid loop program",
			filePath: "../../../examples/in/loop.ft",
			wantErr:  false,
		},
		{
			name:     "union of nominal errors typedef",
			filePath: "../../../examples/in/union_error_types.ft",
			wantErr:  false,
		},
		{
			name:     "union of nominal errors with if-branch narrowing",
			filePath: "../../../examples/in/union_error_narrowing.ft",
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
			c := New(Args{
				Command:  "run",
				FilePath: tt.filePath,
			}, nil)

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
	err = RunGoProgram(outputPath)
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
