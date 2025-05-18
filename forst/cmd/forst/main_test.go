package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExamples(t *testing.T) {
	// Get all example input files
	inputDir := "../../../examples/in"
	outputDir := "../../../examples/out"

	files, err := os.ReadDir(inputDir)
	if err != nil {
		t.Fatalf("Failed to read examples directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".ft") {
			continue
		}

		t.Run(file.Name(), func(t *testing.T) {
			if strings.HasSuffix(file.Name(), ".skip.ft") {
				t.Skip("Skipping test file", file.Name())
				return
			}

			// Get base name without extension
			baseName := strings.TrimSuffix(file.Name(), ".ft")

			// Read input file
			inputPath := filepath.Join(inputDir, file.Name())
			if _, err := os.Stat(inputPath); err != nil {
				t.Fatalf("Input file %s does not exist: %v", inputPath, err)
			}
			// Find expected output file(s)
			outputBasePath := filepath.Join(outputDir, baseName)
			expectedFiles, err := findExpectedOutputFiles(outputBasePath)
			if err != nil {
				t.Fatalf("Failed to find expected output files for %s: %v", baseName, err)
			}

			if len(expectedFiles) == 0 {
				t.Skipf("No expected output files found for %s", baseName)
				return
			}

			// Capture stdout to compare with expected output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run the compiler on the input file
			if err := runCompiler(inputPath); err != nil {
				t.Fatalf("Failed to run compiler: %v", err)
			}
			if err := w.Close(); err != nil {
				t.Fatalf("Failed to close writer: %v", err)
			}
			// Restore stdout
			os.Stdout = oldStdout

			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("failed to copy output: %v", err)
			}
			actualOutput := buf.String()

			// For basic example, compare with the first expected file
			if baseName == "basic" {
				expectedPath := expectedFiles[0]
				expectedContent, err := os.ReadFile(expectedPath)
				if err != nil {
					t.Fatalf("Failed to read expected output file %s: %v", expectedPath, err)
				}

				// Compare actual output with expected content
				compareOutput(t, string(expectedContent), actualOutput)
			} else {
				// For other examples, we might need more complex verification
				// This is a simplified approach - you may need to enhance this
				t.Logf("Generated output for %s:\n%s", baseName, actualOutput)

				// Verify that the output contains key elements from the expected files
				for _, expectedPath := range expectedFiles {
					expectedContent, err := os.ReadFile(expectedPath)
					if err != nil {
						t.Fatalf("Failed to read expected output file %s: %v", expectedPath, err)
					}

					verifyOutputContainsExpectedElements(t, string(expectedContent), actualOutput, expectedPath)
				}
			}
		})
	}
}

// findExpectedOutputFiles returns all .go files in the output directory for a given example
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

// runCompiler executes the compiler on the given input file and returns any error
func runCompiler(inputPath string) error {
	// Create a program instance with args
	args := ProgramArgs{
		command:  "run",
		filePath: inputPath,
	}

	program := &Program{Args: args}
	return program.compileFile()
}

// compareOutput compares the expected and actual output
func compareOutput(t *testing.T, expected, actual string) {
	// Normalize whitespace and line endings
	expected = normalizeString(expected)
	actual = normalizeString(actual)

	if expected != actual {
		t.Errorf("Output mismatch.\nExpected:\n%s\n\nActual:\n%s", expected, actual)
	}
}

// verifyOutputContainsExpectedElements checks if the actual output contains key elements from expected
func verifyOutputContainsExpectedElements(t *testing.T, expected, actual, filePath string) {
	// Extract key elements from the expected output
	// This is a simplified approach - you may need to enhance this
	keyElements := extractKeyElements(expected)

	for _, element := range keyElements {
		if !strings.Contains(actual, element) {
			t.Errorf("Output missing expected element from %s: %s", filePath, element)
		}
	}
}

// extractKeyElements extracts key code elements from Go code
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
	}

	return elements
}

// normalizeString normalizes whitespace and line endings
func normalizeString(s string) string {
	// Replace all whitespace sequences with a single space
	s = strings.Join(strings.Fields(s), " ")
	return s
}
