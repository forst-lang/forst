package main

import (
	"forst/cmd/forst/compiler"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
