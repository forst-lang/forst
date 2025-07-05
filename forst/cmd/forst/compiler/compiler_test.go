package compiler

import (
	"os"
	"path/filepath"
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
			filePath: "../../../../examples/in/basic.ft",
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
