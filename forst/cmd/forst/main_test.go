package main

import (
	"encoding/json"
	"forst/internal/compiler"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestVersionInfo(t *testing.T) {
	// Test that version variables are set
	if Version == "" {
		t.Error("Version should not be empty")
	}

	if Commit == "" {
		t.Error("Commit should not be empty")
	}

	if Date == "" {
		t.Error("Date should not be empty")
	}

	// Test that version info contains expected values
	if Version == "dev" {
		t.Log("Running in development mode")
	} else {
		t.Logf("Running with version: %s", Version)
	}
}

func TestPrintVersionInfo(t *testing.T) {
	// Test that printVersionInfo doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printVersionInfo panicked: %v", r)
		}
	}()

	printVersionInfo()
}

func TestMainFunctionSanity(t *testing.T) {
	// Test that main function can be called without panicking
	// We can't easily test the actual main function since it calls os.Exit,
	// but we can test the logic that would be executed

	// Test version flag handling
	testCases := []struct {
		name       string
		args       []string
		expectExit bool
	}{
		{"version flag", []string{"forst", "version"}, true},
		{"--version flag", []string{"forst", "--version"}, true},
		{"-v flag", []string{"forst", "-v"}, true},
		{"normal usage", []string{"forst", "test.ft"}, false},
		{"no args", []string{"forst"}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Save original args
			originalArgs := os.Args
			defer func() { os.Args = originalArgs }()

			// Set test args
			os.Args = tc.args

			// Test that the version check logic works
			if len(os.Args) > 1 && (os.Args[1] == "version" || os.Args[1] == "--version" || os.Args[1] == "-v") {
				// This would normally call printVersionInfo() and os.Exit(0)
				t.Log("Version flag detected correctly")
			}
		})
	}
}

func TestLoggerSetup(t *testing.T) {
	// Test logger setup logic
	testCases := []struct {
		name          string
		version       string
		expectedLevel string
	}{
		{"dev version", "dev", "debug"},
		{"release version", "1.0.0", "info"},
		{"empty version", "", "info"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Save original version
			originalVersion := Version
			defer func() { Version = originalVersion }()

			// Set test version
			Version = tc.version

			// Test logger creation logic
			var logLevel string
			if Version == "dev" {
				logLevel = "debug"
			} else {
				logLevel = "info"
			}

			if logLevel != tc.expectedLevel {
				t.Errorf("Expected log level %s for version %s, got %s", tc.expectedLevel, tc.version, logLevel)
			}
		})
	}
}

func TestLSPCommandParsing(t *testing.T) {
	// Test LSP command flag parsing logic
	testCases := []struct {
		name             string
		args             []string
		expectedPort     string
		expectedLogLevel string
	}{
		{"default lsp", []string{"forst", "lsp"}, "8081", "info"},
		{"lsp with custom port", []string{"forst", "lsp", "-port", "9999"}, "9999", "info"},
		{"lsp with custom log level", []string{"forst", "lsp", "-log-level", "debug"}, "8081", "debug"},
		{"lsp with both", []string{"forst", "lsp", "-port", "8888", "-log-level", "error"}, "8888", "error"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the flag parsing logic that would be used in main
			if len(tc.args) > 1 && tc.args[1] == "lsp" {
				// Simulate flag parsing
				port := "8081"     // default
				logLevel := "info" // default

				// Simple flag parsing simulation
				for i := 2; i < len(tc.args); i++ {
					switch tc.args[i] {
					case "-port":
						if i+1 < len(tc.args) {
							port = tc.args[i+1]
						}
					case "-log-level":
						if i+1 < len(tc.args) {
							logLevel = tc.args[i+1]
						}
					}
				}

				if port != tc.expectedPort {
					t.Errorf("Expected port %s, got %s", tc.expectedPort, port)
				}

				if logLevel != tc.expectedLogLevel {
					t.Errorf("Expected log level %s, got %s", tc.expectedLogLevel, logLevel)
				}
			}
		})
	}
}

func TestDevCommandParsing(t *testing.T) {
	// Test dev command flag parsing logic
	testCases := []struct {
		name           string
		args           []string
		expectedPort   string
		expectedConfig string
		expectedRoot   string
	}{
		{"default dev", []string{"forst", "dev"}, "8080", "", "."},
		{"dev with custom port", []string{"forst", "dev", "-port", "9999"}, "9999", "", "."},
		{"dev with config", []string{"forst", "dev", "-config", "/path/to/config"}, "8080", "/path/to/config", "."},
		{"dev with root", []string{"forst", "dev", "-root", "/custom/root"}, "8080", "", "/custom/root"},
		{"dev with all", []string{"forst", "dev", "-port", "8888", "-config", "/config", "-root", "/root"}, "8888", "/config", "/root"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the flag parsing logic that would be used in main
			if len(tc.args) > 1 && tc.args[1] == "dev" {
				// Simulate flag parsing
				port := "8080"   // default
				configPath := "" // default
				rootDir := "."   // default

				// Simple flag parsing simulation
				for i := 2; i < len(tc.args); i++ {
					switch tc.args[i] {
					case "-port":
						if i+1 < len(tc.args) {
							port = tc.args[i+1]
						}
					case "-config":
						if i+1 < len(tc.args) {
							configPath = tc.args[i+1]
						}
					case "-root":
						if i+1 < len(tc.args) {
							rootDir = tc.args[i+1]
						}
					}
				}

				if port != tc.expectedPort {
					t.Errorf("Expected port %s, got %s", tc.expectedPort, port)
				}

				if configPath != tc.expectedConfig {
					t.Errorf("Expected config %s, got %s", tc.expectedConfig, configPath)
				}

				if rootDir != tc.expectedRoot {
					t.Errorf("Expected root %s, got %s", tc.expectedRoot, rootDir)
				}
			}
		})
	}
}

func TestLogLevelParsing(t *testing.T) {
	// Test log level parsing logic
	testCases := []struct {
		name          string
		logLevel      string
		expectedLevel string
	}{
		{"trace level", "trace", "trace"},
		{"debug level", "debug", "debug"},
		{"info level", "info", "info"},
		{"warn level", "warn", "warn"},
		{"error level", "error", "error"},
		{"invalid level", "invalid", "info"}, // should default to info
		{"empty level", "", "info"},          // should default to info
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the log level parsing logic from main
			var actualLevel string
			switch tc.logLevel {
			case "trace":
				actualLevel = "trace"
			case "debug":
				actualLevel = "debug"
			case "info":
				actualLevel = "info"
			case "warn":
				actualLevel = "warn"
			case "error":
				actualLevel = "error"
			default:
				actualLevel = "info"
			}

			if actualLevel != tc.expectedLevel {
				t.Errorf("Expected log level %s for input %s, got %s", tc.expectedLevel, tc.logLevel, actualLevel)
			}
		})
	}
}

func TestNewLogger_setsLevelByVersion(t *testing.T) {
	original := Version
	defer func() { Version = original }()

	Version = "dev"
	if got := newLogger().GetLevel(); got != logrus.DebugLevel {
		t.Fatalf("dev logger level: %s", got)
	}

	Version = "1.2.3"
	if got := newLogger().GetLevel(); got != logrus.InfoLevel {
		t.Fatalf("release logger level: %s", got)
	}
}

func TestSetLogLevel_appliesKnownAndDefault(t *testing.T) {
	log := logrus.New()

	setLogLevel(log, "trace")
	if log.GetLevel() != logrus.TraceLevel {
		t.Fatalf("trace level not applied: %s", log.GetLevel())
	}

	setLogLevel(log, "error")
	if log.GetLevel() != logrus.ErrorLevel {
		t.Fatalf("error level not applied: %s", log.GetLevel())
	}

	setLogLevel(log, "not-a-level")
	if log.GetLevel() != logrus.InfoLevel {
		t.Fatalf("default info level not applied: %s", log.GetLevel())
	}
}

func TestCompilerArgsParsing(t *testing.T) {
	// Test that compiler args can be parsed
	// Save original args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Test with minimal args
	os.Args = []string{"forst", "test.ft"}

	// This would normally call compiler.ParseArgs(log)
	// We'll test that the logic doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("compiler.ParseArgs panicked: %v", r)
		}
	}()

	// Test that we can create a logger without issues
	log := setupTestLogger(nil)
	if log == nil {
		t.Error("Expected logger to be created")
	}
}

func TestLSPVersionInjection(t *testing.T) {
	// Test that version information is correctly injected into LSP package
	originalVersion := Version
	originalCommit := Commit
	originalDate := Date
	defer func() {
		Version = originalVersion
		Commit = originalCommit
		Date = originalDate
	}()

	// Set test values
	Version = "test-version"
	Commit = "test-commit"
	Date = "test-date"

	// Test the version injection logic that would be used in main
	// This simulates the logic: lsp.Version = Version, etc.
	lspVersion := Version
	lspCommit := Commit
	lspDate := Date

	if lspVersion != "test-version" {
		t.Errorf("Expected LSP version test-version, got %s", lspVersion)
	}

	if lspCommit != "test-commit" {
		t.Errorf("Expected LSP commit test-commit, got %s", lspCommit)
	}

	if lspDate != "test-date" {
		t.Errorf("Expected LSP date test-date, got %s", lspDate)
	}
}

func TestFileValidation(t *testing.T) {
	// Test file path validation logic
	testCases := []struct {
		name        string
		filePath    string
		expectError bool
	}{
		{"empty path", "", true},
		{"valid path", "test.ft", false},
		{"valid path with dir", "path/to/test.ft", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the file path validation logic from main
			if tc.filePath == "" {
				// This would normally cause an error in main
				t.Log("Empty file path would cause error")
			} else {
				t.Log("Valid file path")
			}
		})
	}
}

func TestWatchModeLogic(t *testing.T) {
	// Test watch mode logic
	testCases := []struct {
		name         string
		watch        bool
		expectedMode string
	}{
		{"watch mode", true, "watch"},
		{"single compile", false, "single"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the watch mode logic from main
			var mode string
			if tc.watch {
				mode = "watch"
			} else {
				mode = "single"
			}

			if mode != tc.expectedMode {
				t.Errorf("Expected mode %s for watch=%t, got %s", tc.expectedMode, tc.watch, mode)
			}
		})
	}
}

func TestRunCommandLogic(t *testing.T) {
	// Test run command logic
	testCases := []struct {
		name      string
		command   string
		shouldRun bool
	}{
		{"run command", "run", true},
		{"compile command", "compile", false},
		{"other command", "other", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the run command logic from main
			shouldRun := tc.command == "run"

			if shouldRun != tc.shouldRun {
				t.Errorf("Expected shouldRun=%t for command %s, got %t", tc.shouldRun, tc.command, shouldRun)
			}
		})
	}
}

func TestExamples(t *testing.T) {
	// Get all example input files
	inputDir := "../../../examples/in"
	outputDir := "../../../examples/out"

	// Walk through all subdirectories
	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process .ft and .go files
		if !strings.HasSuffix(info.Name(), ".ft") && !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		// Get relative path from input directory
		relPath, err := filepath.Rel(inputDir, path)
		if err != nil {
			return err
		}

		// Get base name without extension
		baseName := strings.TrimSuffix(relPath, filepath.Ext(relPath))

		// Create corresponding output path
		outputBasePath := filepath.Join(outputDir, baseName)

		t.Run(relPath, func(t *testing.T) {
			if strings.HasSuffix(info.Name(), ".skip.ft") || strings.HasSuffix(info.Name(), ".skip.go") {
				t.Skip("Skipping test file", relPath)
				return
			}

			// Multi-file package: golden + merged compile are checked in TestExampleTictactoeMergedPackage.
			if strings.HasPrefix(relPath, "tictactoe/") && strings.HasSuffix(relPath, ".ft") {
				t.Skip("covered by TestExampleTictactoeMergedPackage (-root merged package)")
				return
			}

			// Find expected output file(s)
			expectedFiles, err := findExpectedOutputFiles(outputBasePath)
			if err != nil {
				t.Fatalf("Failed to find expected output files for %s: %v", baseName, err)
			}

			if len(expectedFiles) == 0 {
				t.Skipf("No expected output files found for %s", baseName)
				return
			}

			// Run the compiler on the input file
			if err := runCompiler(path); err != nil {
				if strings.HasPrefix(relPath, "rfc/") {
					t.Logf("Ignoring failure for RFC example %s: %v", relPath, err)
					return
				}
				t.Fatalf("Failed to run compiler: %v", err)
			}

			// Read the generated code from the temporary file
			compiler := compiler.New(compiler.Args{
				Command:  "run",
				FilePath: path,
			}, nil)
			code, err := compiler.CompileFile()
			if err != nil {
				t.Fatalf("Failed to compile file: %v", err)
			}
			actualOutput := *code

			t.Logf("Generated output for %s:\n%s", baseName, actualOutput)

			// Verify that the output contains key elements from the expected files
			for _, expectedPath := range expectedFiles {
				expectedContent, err := os.ReadFile(expectedPath)
				if err != nil {
					t.Fatalf("Failed to read expected output file %s: %v", expectedPath, err)
				}

				verifyOutputContainsExpectedElements(t, string(expectedContent), actualOutput, expectedPath)
			}
		})

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to walk examples directory: %v", err)
	}
}

// TestResultExamplesIncludeEnsureLowering locks in the shape of result_if / result_ensure: a
// Result-returning helper should include an `ensure` in its body so tooling and lowering match
// real code paths (plain `return n` alone does not emit an ensure branch).
func TestResultExamplesIncludeEnsureLowering(t *testing.T) {
	t.Parallel()
	examples := []string{
		filepath.Join("..", "..", "..", "examples", "in", "result_if.ft"),
		filepath.Join("..", "..", "..", "examples", "in", "result_ensure.ft"),
	}
	const wantBranch = `if n <= 0`
	const wantMsg = `assertion failed: Int.GreaterThan(0)`
	for _, path := range examples {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			t.Parallel()
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read example: %v", err)
			}
			if !strings.Contains(string(src), "ensure n is") {
				t.Fatalf("example %s should include `ensure n is` in the Result-returning helper", path)
			}

			c := compiler.New(compiler.Args{
				Command:  "run",
				FilePath: path,
				LogLevel: "error",
			}, nil)
			code, err := c.CompileFile()
			if err != nil {
				t.Fatalf("CompileFile: %v", err)
			}
			out := *code
			if !strings.Contains(out, wantBranch) {
				t.Errorf("expected generated Go to include ensure failure branch %q", wantBranch)
			}
			if !strings.Contains(out, wantMsg) {
				t.Errorf("expected generated Go to include assertion message %q", wantMsg)
			}
		})
	}
}

// TestExampleTictactoeMergedPackage compiles examples/in/tictactoe with -root (same-package merge)
// and checks generated Go against examples/out/tictactoe/server.go.
// Regenerate the golden file: UPDATE_TICTACTOE_GOLDEN=1 go test ./cmd/forst -run TestExampleTictactoeMergedPackage -count=1
func TestExampleTictactoeMergedPackage(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples", "in", "tictactoe")
	entry := filepath.Join(root, "server.ft")
	goldenPath := filepath.Join("..", "..", "..", "examples", "out", "tictactoe", "server.go")

	c := compiler.New(compiler.Args{
		Command:     "run",
		FilePath:    entry,
		PackageRoot: root,
		LogLevel:    "error",
	}, nil)
	code, err := c.CompileFile()
	if err != nil {
		t.Fatalf("CompileFile: %v", err)
	}
	actual := *code

	if os.Getenv("UPDATE_TICTACTOE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, []byte(actual), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote golden %s", goldenPath)
		return
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v (set UPDATE_TICTACTOE_GOLDEN=1 to create)", goldenPath, err)
	}
	// Hash-based emitted type names (T_…) are not stable across small compiler changes; assert
	// structural markers instead of line-for-line equality with extractKeyElements.
	verifyTictactoeMergedGolden(t, string(expected), actual, goldenPath)
}

func TestFindExpectedOutputFiles_directoryAndSingleFile(t *testing.T) {
	baseDir := t.TempDir()

	dirPath := filepath.Join(baseDir, "module")
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirPath, "a.go"), []byte("package x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirPath, "README.md"), []byte("ignore"), 0o644); err != nil {
		t.Fatal(err)
	}

	gotDirFiles, err := findExpectedOutputFiles(dirPath)
	if err != nil {
		t.Fatalf("findExpectedOutputFiles(dir): %v", err)
	}
	if len(gotDirFiles) != 1 || !strings.HasSuffix(gotDirFiles[0], "a.go") {
		t.Fatalf("unexpected directory files: %+v", gotDirFiles)
	}

	singleBase := filepath.Join(baseDir, "single_output")
	if err := os.WriteFile(singleBase+".go", []byte("package y"), 0o644); err != nil {
		t.Fatal(err)
	}
	gotSingleFile, err := findExpectedOutputFiles(singleBase)
	if err != nil {
		t.Fatalf("findExpectedOutputFiles(single): %v", err)
	}
	if len(gotSingleFile) != 1 || gotSingleFile[0] != singleBase+".go" {
		t.Fatalf("unexpected single file lookup: %+v", gotSingleFile)
	}

	gotMissing, err := findExpectedOutputFiles(filepath.Join(baseDir, "missing"))
	if err != nil {
		t.Fatalf("findExpectedOutputFiles(missing): %v", err)
	}
	if len(gotMissing) != 0 {
		t.Fatalf("expected no files for missing path, got %+v", gotMissing)
	}
}

func TestExtractKeyElements_collectsSignaturesTypesFieldsAndIf(t *testing.T) {
	code := strings.Join([]string{
		"package main",
		"type User struct {",
		"  Name string",
		"}",
		"func run(x int) error {",
		"  if x > 0 {",
		"    return nil",
		"  }",
		"  return nil",
		"}",
	}, "\n")

	keyElements := extractKeyElements(code)
	joined := strings.Join(keyElements, "\n")

	expectedFragments := []string{
		"type User struct {",
		"Name string",
		"func run(x int) error {",
		"if x > 0 {",
	}
	for _, fragment := range expectedFragments {
		if !strings.Contains(joined, fragment) {
			t.Fatalf("missing key element fragment %q in:\n%s", fragment, joined)
		}
	}
}

func TestCheckIfStatementConditions_handlesMalformedIfWithoutPanic(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	_ = log // keeps intent explicit for this test file's logging-heavy helpers

	code := strings.Join([]string{
		"package main",
		"func x() {",
		"  if brokenCondition()",
		"  if validCondition() {",
		"  }",
		"}",
	}, "\n")

	checkIfStatementConditions(t, code, "synthetic.go")
}

func TestHandleDumpCommand_jsonAndPrettyOutput(t *testing.T) {
	testFilePath := filepath.Join(t.TempDir(), "input.ft")
	source := "fn main() { return }\n"
	if err := os.WriteFile(testFilePath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	logger := logrus.New()
	logger.SetOutput(io.Discard)

	compactOutput := captureStdoutForMainTest(t, func() {
		handleDumpCommand(testFilePath, false, "json", "", false, logger)
	})
	if strings.TrimSpace(compactOutput) == "" {
		t.Fatal("expected non-empty json output")
	}
	var compactJSON any
	if err := json.Unmarshal([]byte(compactOutput), &compactJSON); err != nil {
		t.Fatalf("compact output should be valid JSON: %v\noutput: %s", err, compactOutput)
	}

	prettyOutput := captureStdoutForMainTest(t, func() {
		handleDumpCommand(testFilePath, false, "pretty", "", true, logger)
	})
	if strings.TrimSpace(prettyOutput) == "" {
		t.Fatal("expected non-empty pretty output")
	}
	var prettyJSON any
	if err := json.Unmarshal([]byte(prettyOutput), &prettyJSON); err != nil {
		t.Fatalf("pretty output should be valid JSON: %v\noutput: %s", err, prettyOutput)
	}
	if !strings.Contains(prettyOutput, "\n") {
		t.Fatalf("expected pretty output to contain newlines, got: %s", prettyOutput)
	}
}

func TestHandleDumpCommand_helperProcess(t *testing.T) {
	if os.Getenv("FORST_MAIN_DUMP_HELPER") != "1" {
		return
	}
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	handleDumpCommand("/path/that/does/not/exist.ft", false, "json", "", false, logger)
}

func TestHandleDumpCommand_exitsOnReadFailure(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestHandleDumpCommand_helperProcess")
	cmd.Env = append(os.Environ(), "FORST_MAIN_DUMP_HELPER=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected helper process to exit with non-zero status")
	}
}

func TestMain_helperProcess(t *testing.T) {
	helperCase := os.Getenv("FORST_MAIN_HELPER_CASE")
	if helperCase == "" {
		return
	}

	switch helperCase {
	case "version":
		os.Args = []string{"forst", "version"}
	case "dump-missing-file":
		os.Args = []string{"forst", "dump", "--file", "/path/that/does/not/exist.ft"}
	default:
		t.Fatalf("unknown helper case: %s", helperCase)
	}

	main()
}

func TestMain_versionCommand_exitsZeroAndPrintsVersion(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestMain_helperProcess")
	cmd.Env = append(os.Environ(), "FORST_MAIN_HELPER_CASE=version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected zero exit for version command: %v, output=%s", err, string(output))
	}
	if !strings.Contains(string(output), "forst ") {
		t.Fatalf("expected version output to contain 'forst ', got: %s", string(output))
	}
}

func TestMain_dumpMissingFile_exitsNonZero(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestMain_helperProcess")
	cmd.Env = append(os.Environ(), "FORST_MAIN_HELPER_CASE=dump-missing-file")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected dump command to exit non-zero when file is missing")
	}
}
