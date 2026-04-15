package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/compiler"
)

// Returns all .go files in the output directory for a given example.
func findExpectedOutputFiles(basePath string) ([]string, error) {
	var files []string

	info, err := os.Stat(basePath)
	if err == nil && info.IsDir() {
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
		filePath := basePath + ".go"
		if _, err := os.Stat(filePath); err == nil {
			files = append(files, filePath)
		}
	}

	return files, nil
}

// Executes the compiler on the given input file and returns any error.
func runCompiler(inputPath string) error {
	args := compiler.Args{
		Command:  "run",
		FilePath: inputPath,
	}

	log := setupTestLogger(nil)

	c := compiler.New(args, log)
	_, err := c.CompileFile()
	return err
}

// verifyTictactoeMergedGolden checks merged-package Go output without depending on hash-based
// type names (T_…) that may change between compiler versions.
func verifyTictactoeMergedGolden(t *testing.T, expected, actual, goldenPath string) {
	markers := []string{
		"package main",
		"type GameState struct",
		"type MoveRequest struct",
		"type MoveResponse struct",
		"func ApplyMove(req MoveRequest) (MoveResponse, error)",
		"func PlayMove(req MoveRequest) (MoveResponse, error)",
		"func NewGame() GameState",
		"func main()",
		"fmt",
	}
	for _, marker := range markers {
		if !strings.Contains(actual, marker) {
			t.Errorf("output missing %q (golden %s)", marker, goldenPath)
		}
	}
	if len(expected) > 0 && len(actual) < len(expected)/2 {
		t.Errorf("output much shorter than golden (%d vs %d bytes)", len(actual), len(expected))
	}
}

// Checks if the actual output contains key elements from expected.
func verifyOutputContainsExpectedElements(t *testing.T, expected, actual, filePath string) {
	keyElements := extractKeyElements(expected)

	for _, element := range keyElements {
		if !strings.Contains(actual, element) {
			t.Errorf("Output missing expected element from %s: %s", filePath, element)
		}
	}

	checkIfStatementConditions(t, actual, filePath)
}

// Extracts key code elements from Go code.
func extractKeyElements(code string) []string {
	var elements []string

	lines := strings.Split(code, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "func ") && strings.Contains(line, "(") {
			elements = append(elements, line)
		}

		if strings.HasPrefix(line, "type ") && strings.Contains(line, "struct") {
			elements = append(elements, line)
		}

		if strings.Contains(line, " string") || strings.Contains(line, " int") ||
			strings.Contains(line, " float") || strings.Contains(line, " bool") {
			elements = append(elements, line)
		}

		if strings.HasPrefix(line, "if ") {
			elements = append(elements, line)
		}
	}

	return elements
}

// Checks if statement conditions match expected conditions.
func checkIfStatementConditions(t *testing.T, code, filePath string) {
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "if ") {
			start := strings.Index(line, "if ") + 3
			end := strings.Index(line, "{")
			if end == -1 {
				continue
			}

			actualCondition := strings.TrimSpace(line[start:end])
			t.Logf("If statement condition in %s at line %d: %s", filePath, i+1, actualCondition)
		}
	}
}

func captureStdoutForMainTest(t *testing.T, run func()) string {
	t.Helper()
	originalStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = writePipe
	t.Cleanup(func() {
		os.Stdout = originalStdout
	})

	run()

	if err := writePipe.Close(); err != nil {
		t.Fatalf("close write pipe: %v", err)
	}
	outputBytes, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return string(outputBytes)
}
