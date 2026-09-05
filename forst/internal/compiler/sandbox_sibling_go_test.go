package compiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateTempOutputFiles_copiesSamePackageHandwrittenGo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	helperContent := `package main

func AddInts(a, b int) int {
	return a + b
}
`
	if err := os.WriteFile(filepath.Join(dir, "helpers.go"), []byte(helperContent), 0o644); err != nil {
		t.Fatal(err)
	}

	mainFtPath := filepath.Join(dir, "main.ft")
	mainCode := `package main

func main() {
	_ = AddInts(2, 3)
}
`
	if err := os.WriteFile(mainFtPath, []byte(mainCode), 0o644); err != nil {
		t.Fatal(err)
	}

	outputPath, err := CreateTempOutputFilesForEntry(mainCode, "", "", nil, nil, dir, mainFtPath)
	if err != nil {
		t.Fatalf("CreateTempOutputFilesForEntry: %v", err)
	}

	tempDir := filepath.Dir(outputPath)
	copiedHelper := filepath.Join(tempDir, "helpers.go")
	if _, err := os.Stat(copiedHelper); err != nil {
		t.Fatalf("expected helpers.go to be copied into sandbox tempDir %s, got err: %v", tempDir, err)
	}

	// Verify the sandbox compiles via go build
	if err := BuildGoProgram(mainCode, "", "", nil, dir); err != nil {
		t.Fatalf("BuildGoProgram failed for sandbox with sibling .go: %v", err)
	}
}
