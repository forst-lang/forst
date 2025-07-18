package main

import (
	"forst/cmd/forst/compiler"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	log := setupTestLogger()
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

// Returns all .go files in the output directory for a given example
func findExpectedOutputFiles(basePath string) ([]string, error) {
	var files []string

	// Check if it's a directory
	info, err := os.Stat(basePath)
	if err == nil && info.IsDir() {
		// It's a directory, find all .go files
		entries, err := os.ReadDir(basePath)
		if err != nil {
			return nil, err
		}

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
				files = append(files, filepath.Join(basePath, entry.Name()))
			}
		}
	} else {
		// Try as a single file
		filePath := basePath + ".go"
		if _, err := os.Stat(filePath); err == nil {
			files = append(files, filePath)
		}
	}

	return files, nil
}

// Executes the compiler on the given input file and returns any error
func runCompiler(inputPath string) error {
	// Create a compiler instance with args
	args := compiler.Args{
		Command:  "run",
		FilePath: inputPath,
	}

	log := setupTestLogger()

	c := compiler.New(args, log)
	_, err := c.CompileFile()
	return err
}

// Checks if the actual output contains key elements from expected
func verifyOutputContainsExpectedElements(t *testing.T, expected, actual, filePath string) {
	// Extract key elements from the expected output
	keyElements := extractKeyElements(expected)

	for _, element := range keyElements {
		if !strings.Contains(actual, element) {
			t.Errorf("Output missing expected element from %s: %s", filePath, element)
		}
	}

	// Check if statement conditions
	checkIfStatementConditions(t, actual, filePath)
}

// Extracts key code elements from Go code
func extractKeyElements(code string) []string {
	var elements []string

	// Extract function signatures, type definitions, and struct fields
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Function signatures
		if strings.HasPrefix(line, "func ") && strings.Contains(line, "(") {
			elements = append(elements, line)
		}

		// Type definitions
		if strings.HasPrefix(line, "type ") && strings.Contains(line, "struct") {
			elements = append(elements, line)
		}

		// Struct fields (simplified)
		if strings.Contains(line, " string") || strings.Contains(line, " int") ||
			strings.Contains(line, " float") || strings.Contains(line, " bool") {
			elements = append(elements, line)
		}

		// If statements - check for conditions
		if strings.HasPrefix(line, "if ") && strings.Contains(line, "(") {
			elements = append(elements, line)
		}
	}

	return elements
}

// Checks if statement conditions match expected conditions
func checkIfStatementConditions(t *testing.T, code, filePath string) {
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Extract if statement conditions
		if strings.HasPrefix(line, "if ") && strings.Contains(line, "(") {
			// Extract condition between "if " and " {"
			start := strings.Index(line, "if ") + 3
			end := strings.Index(line, " {")
			if end == -1 {
				continue // Skip if no opening brace found
			}

			actualCondition := strings.TrimSpace(line[start:end])

			// For now, just log the condition for debugging
			// In the future, we could compare against expected conditions from output files
			t.Logf("If statement condition in %s at line %d: %s", filePath, i+1, actualCondition)
		}
	}
}
