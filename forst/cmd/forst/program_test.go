package main

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
			program := NewProgram(ProgramArgs{
				command:  "run",
				filePath: tt.filePath,
			})

			code, err := program.compileFile()
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

	outputPath, err := createTempOutputFile(testCode)
	if err != nil {
		t.Fatalf("createTempOutputFile() error = %v", err)
	}

	defer func() {
		err := os.RemoveAll(filepath.Dir(outputPath))
		if err != nil {
			t.Errorf("Failed to remove temporary output file: %v", err)
		}
	}()

	// Verify the file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("createTempOutputFile() did not create the output file")
	}

	// Verify the content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != testCode {
		t.Errorf("createTempOutputFile() content = %v, want %v", string(content), testCode)
	}
}

func TestRunGoProgram(t *testing.T) {
	// Create a temporary test program
	testCode := "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}"
	outputPath, err := createTempOutputFile(testCode)
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
	err = runGoProgram(outputPath)
	if err != nil {
		t.Errorf("runGoProgram() error = %v", err)
	}
}
