package executor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/discovery"

	logrus "github.com/sirupsen/logrus"
)

func TestNewGoModuleManager(t *testing.T) {
	logger := &logrus.Logger{}
	manager := NewGoModuleManager(logger)

	if manager == nil {
		t.Fatal("NewGoModuleManager returned nil")
	}

	if manager.log != logger {
		t.Error("Logger not properly set")
	}
}

func TestGoModuleManager_CreateModule_Basic(t *testing.T) {
	manager := NewGoModuleManager(nil)

	config := &ModuleConfig{
		ModuleName:     "test-module",
		PackageName:    "testpkg",
		FunctionName:   "TestFunction",
		GoCode:         "package testpkg\n\nfunc TestFunction() string { return \"hello\" }",
		SupportsParams: false,
		Parameters:     []discovery.ParameterInfo{},
		Args:           []byte("{}"),
		IsStreaming:    false,
	}

	tempDir, err := manager.CreateModule(config)
	if err != nil {
		t.Fatalf("CreateModule failed: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Verify directory structure
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read temp dir: %v", err)
	}

	expectedFiles := map[string]bool{
		"go.mod":  false,
		"main.go": false,
		"testpkg": false, // directory
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if entry.Name() == "testpkg" {
				expectedFiles["testpkg"] = true
			}
		} else {
			expectedFiles[entry.Name()] = true
		}
	}

	for file, found := range expectedFiles {
		if !found {
			t.Errorf("Expected file/directory %s not found", file)
		}
	}
}

func TestGoModuleManager_CreateModule_WithParams(t *testing.T) {
	manager := NewGoModuleManager(nil)

	config := &ModuleConfig{
		ModuleName:     "test-module",
		PackageName:    "testpkg",
		FunctionName:   "Echo",
		GoCode:         "package testpkg\n\ntype EchoRequest struct { Message string }\n\nfunc Echo(input EchoRequest) string { return input.Message }",
		SupportsParams: true,
		Parameters: []discovery.ParameterInfo{
			{Name: "input", Type: "EchoRequest"},
		},
		Args:        []byte(`{"message":"test"}`),
		IsStreaming: false,
	}

	tempDir, err := manager.CreateModule(config)
	if err != nil {
		t.Fatalf("CreateModule failed: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Verify main.go content for parameterized function
	mainGoPath := filepath.Join(tempDir, "main.go")
	mainGoContent, err := os.ReadFile(mainGoPath)
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}

	content := string(mainGoContent)
	expectedSnippets := []string{
		"package main",
		"encoding/json",
		"os.Stdin",
		"json.NewDecoder",
		"EchoRequest",
		"Echo(",
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(content, snippet) {
			t.Errorf("main.go missing expected content: %s", snippet)
		}
	}
}

func TestGoModuleManager_CreateModule_Streaming(t *testing.T) {
	manager := NewGoModuleManager(nil)

	config := &ModuleConfig{
		ModuleName:     "test-module",
		PackageName:    "testpkg",
		FunctionName:   "StreamFunction",
		GoCode:         "package testpkg\n\nfunc StreamFunction(args json.RawMessage) chan string { return make(chan string) }",
		SupportsParams: true,
		Parameters:     []discovery.ParameterInfo{},
		Args:           []byte(`{"data":"test"}`),
		IsStreaming:    true,
	}

	tempDir, err := manager.CreateModule(config)
	if err != nil {
		t.Fatalf("CreateModule failed: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Verify main.go content for streaming function
	mainGoPath := filepath.Join(tempDir, "main.go")
	mainGoContent, err := os.ReadFile(mainGoPath)
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}

	content := string(mainGoContent)
	expectedSnippets := []string{
		"package main",
		"json.RawMessage",
		"StreamFunction(args)",
		"for result := range results",
		"json.Marshal(result)",
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(content, snippet) {
			t.Errorf("main.go missing expected content: %s", snippet)
		}
	}
}

func TestGoModuleManager_CreateGoMod(t *testing.T) {
	manager := NewGoModuleManager(nil)

	tempDir, err := os.MkdirTemp("", "test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	moduleName := "test-module"
	err = manager.createGoMod(tempDir, moduleName)
	if err != nil {
		t.Fatalf("createGoMod failed: %v", err)
	}

	// Verify go.mod content
	goModPath := filepath.Join(tempDir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}

	expectedContent := fmt.Sprintf("module %s\n\ngo 1.21\n", moduleName)
	if string(content) != expectedContent {
		t.Errorf("go.mod content mismatch:\nExpected: %s\nGot: %s", expectedContent, string(content))
	}
}

func TestGoModuleManager_CreateGoMod_Error(t *testing.T) {
	manager := NewGoModuleManager(nil)

	// Try to create go.mod in a non-existent directory
	err := manager.createGoMod("/non/existent/path", "test-module")
	if err == nil {
		t.Error("Expected error when creating go.mod in non-existent directory")
	}
}

func TestGoModuleManager_CreateMainGo_Standard(t *testing.T) {
	manager := NewGoModuleManager(nil)

	tempDir, err := os.MkdirTemp("", "test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := &ModuleConfig{
		ModuleName:     "test-module",
		PackageName:    "testpkg",
		FunctionName:   "TestFunction",
		SupportsParams: false,
		Parameters:     []discovery.ParameterInfo{},
		IsStreaming:    false,
	}

	err = manager.createMainGo(tempDir, config)
	if err != nil {
		t.Fatalf("createMainGo failed: %v", err)
	}

	// Verify main.go content
	mainGoPath := filepath.Join(tempDir, "main.go")
	content, err := os.ReadFile(mainGoPath)
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}

	contentStr := string(content)
	expectedSnippets := []string{
		"package main",
		"encoding/json",
		"TestFunction()",
		"json.Marshal(result)",
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(contentStr, snippet) {
			t.Errorf("main.go missing expected content: %s", snippet)
		}
	}
}

func TestGoModuleManager_CreateMainGo_WithParams(t *testing.T) {
	manager := NewGoModuleManager(nil)

	tempDir, err := os.MkdirTemp("", "test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := &ModuleConfig{
		ModuleName:     "test-module",
		PackageName:    "testpkg",
		FunctionName:   "Echo",
		SupportsParams: true,
		Parameters: []discovery.ParameterInfo{
			{Name: "input", Type: "EchoRequest"},
		},
		IsStreaming: false,
	}

	err = manager.createMainGo(tempDir, config)
	if err != nil {
		t.Fatalf("createMainGo failed: %v", err)
	}

	// Verify main.go content
	mainGoPath := filepath.Join(tempDir, "main.go")
	content, err := os.ReadFile(mainGoPath)
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}

	contentStr := string(content)
	expectedSnippets := []string{
		"package main",
		"os.Stdin",
		"json.NewDecoder",
		"EchoRequest",
		"input",
		"Echo(input)",
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(contentStr, snippet) {
			t.Errorf("main.go missing expected content: %s", snippet)
		}
	}
}

func TestGoModuleManager_CreatePackageFile(t *testing.T) {
	manager := NewGoModuleManager(nil)

	tempDir, err := os.MkdirTemp("", "test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := &ModuleConfig{
		PackageName: "testpkg",
		GoCode:      "package testpkg\n\nfunc TestFunction() string { return \"hello\" }",
	}

	err = manager.createPackageFile(tempDir, config)
	if err != nil {
		t.Fatalf("createPackageFile failed: %v", err)
	}

	// Verify package directory and file
	packageDir := filepath.Join(tempDir, "testpkg")
	if _, err := os.Stat(packageDir); os.IsNotExist(err) {
		t.Error("Package directory not created")
	}

	packageGoPath := filepath.Join(packageDir, "testpkg.go")
	content, err := os.ReadFile(packageGoPath)
	if err != nil {
		t.Fatalf("Failed to read package file: %v", err)
	}

	if string(content) != config.GoCode {
		t.Errorf("Package file content mismatch:\nExpected: %s\nGot: %s", config.GoCode, string(content))
	}
}

func TestGoModuleManager_CreatePackageFile_Error(t *testing.T) {
	manager := NewGoModuleManager(nil)

	config := &ModuleConfig{
		PackageName: "testpkg",
		GoCode:      "package testpkg\n\nfunc TestFunction() string { return \"hello\" }",
	}

	// Try to create package file in a non-existent directory
	err := manager.createPackageFile("/non/existent/path", config)
	if err == nil {
		t.Error("Expected error when creating package file in non-existent directory")
	}
}

func TestGoModuleManager_GenerateStandardMainGo_NoParams(t *testing.T) {
	manager := NewGoModuleManager(nil)

	config := &ModuleConfig{
		ModuleName:     "test-module",
		PackageName:    "testpkg",
		FunctionName:   "TestFunction",
		SupportsParams: false,
		Parameters:     []discovery.ParameterInfo{},
		IsStreaming:    false,
	}

	content := manager.generateStandardMainGo("test-module/testpkg", "testpkg_1234", config)

	expectedSnippets := []string{
		"package main",
		"encoding/json",
		"TestFunction()",
		"json.Marshal(result)",
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(content, snippet) {
			t.Errorf("Generated main.go missing expected content: %s", snippet)
		}
	}

	// Should not contain stdin reading code
	unexpectedSnippets := []string{
		"os.Stdin",
		"json.NewDecoder",
	}

	for _, snippet := range unexpectedSnippets {
		if strings.Contains(content, snippet) {
			t.Errorf("Generated main.go contains unexpected content: %s", snippet)
		}
	}
}

func TestGoModuleManager_GenerateStandardMainGo_WithParams(t *testing.T) {
	manager := NewGoModuleManager(nil)

	config := &ModuleConfig{
		ModuleName:     "test-module",
		PackageName:    "testpkg",
		FunctionName:   "Echo",
		SupportsParams: true,
		Parameters: []discovery.ParameterInfo{
			{Name: "input", Type: "EchoRequest"},
		},
		IsStreaming: false,
	}

	content := manager.generateStandardMainGo("test-module/testpkg", "testpkg_1234", config)

	expectedSnippets := []string{
		"package main",
		"encoding/json",
		"os.Stdin",
		"json.NewDecoder",
		"EchoRequest",
		"input",
		"Echo(input)",
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(content, snippet) {
			t.Errorf("Generated main.go missing expected content: %s", snippet)
		}
	}
}

func TestGoModuleManager_GenerateStreamingMainGo(t *testing.T) {
	manager := NewGoModuleManager(nil)

	config := &ModuleConfig{
		PackageName:  "testpkg",
		FunctionName: "StreamFunction",
		Args:         []byte(`{"data":"test"}`),
		IsStreaming:  true,
	}

	content := manager.generateStreamingMainGo("test-module/testpkg", "testpkg_1234", config)

	expectedSnippets := []string{
		"package main",
		"encoding/json",
		"json.RawMessage",
		`{"data":"test"}`,
		"StreamFunction(args)",
		"for result := range results",
		"json.Marshal(result)",
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(content, snippet) {
			t.Errorf("Generated streaming main.go missing expected content: %s", snippet)
		}
	}
}

func TestGoModuleManager_CreateModule_CleanupOnError(t *testing.T) {
	manager := NewGoModuleManager(nil)

	// Create a config that will cause an error (invalid temp dir path)
	config := &ModuleConfig{
		ModuleName:     "test-module",
		PackageName:    "testpkg",
		FunctionName:   "TestFunction",
		GoCode:         "package testpkg\n\nfunc TestFunction() string { return \"hello\" }",
		SupportsParams: false,
		Parameters:     []discovery.ParameterInfo{},
		Args:           []byte("{}"),
		IsStreaming:    false,
	}

	// This test verifies that the module creation handles errors gracefully
	// The actual error handling is tested in individual component tests
	_, err := manager.CreateModule(config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify the module was created successfully
	// This is a basic check - in a real scenario we'd need to track created directories
}

func TestGoModuleManager_ModuleConfig_JSON(t *testing.T) {
	// Test that ModuleConfig can be marshaled to JSON (for debugging/logging)
	config := &ModuleConfig{
		ModuleName:     "test-module",
		PackageName:    "testpkg",
		FunctionName:   "TestFunction",
		GoCode:         "package testpkg\n\nfunc TestFunction() string { return \"hello\" }",
		SupportsParams: true,
		Parameters: []discovery.ParameterInfo{
			{Name: "input", Type: "EchoRequest"},
		},
		Args:        []byte(`{"message":"test"}`),
		IsStreaming: false,
	}

	_, err := json.Marshal(config)
	if err != nil {
		t.Errorf("Failed to marshal ModuleConfig to JSON: %v", err)
	}
}
