package discovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"unicode"

	"forst/internal/ast"
)

// MockConfig implements configiface.ForstConfigIface for testing
type MockConfig struct {
	files []string
	err   error
}

func (m *MockConfig) FindForstFiles(rootDir string) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.files, nil
}

// MockLogger implements Logger interface for testing
type MockLogger struct {
	debugMsgs []string
	infoMsgs  []string
	warnMsgs  []string
	errorMsgs []string
	traceMsgs []string
}

func (m *MockLogger) Debugf(format string, args ...interface{}) {
	m.debugMsgs = append(m.debugMsgs, format)
}

func (m *MockLogger) Infof(format string, args ...interface{}) {
	m.infoMsgs = append(m.infoMsgs, format)
}

func (m *MockLogger) Warnf(format string, args ...interface{}) {
	m.warnMsgs = append(m.warnMsgs, format)
}

func (m *MockLogger) Errorf(format string, args ...interface{}) {
	m.errorMsgs = append(m.errorMsgs, format)
}

func (m *MockLogger) Tracef(format string, args ...interface{}) {
	m.traceMsgs = append(m.traceMsgs, format)
}

func TestNewDiscoverer(t *testing.T) {
	logger := &MockLogger{}
	config := &MockConfig{}

	discoverer := NewDiscoverer("/test/path", logger, config)

	if discoverer == nil {
		t.Fatal("NewDiscoverer returned nil")
	}

	if discoverer.rootDir != "/test/path" {
		t.Errorf("Expected rootDir '/test/path', got '%s'", discoverer.rootDir)
	}

	if discoverer.log != logger {
		t.Error("Logger not properly set")
	}

	if discoverer.config != config {
		t.Error("Config not properly set")
	}
}

func TestDiscoverer_FindForstFiles_Success(t *testing.T) {
	logger := &MockLogger{}
	config := &MockConfig{
		files: []string{"/test/file1.ft", "/test/file2.ft"},
	}

	discoverer := NewDiscoverer("/test/path", logger, config)

	files, err := discoverer.findForstFiles()
	if err != nil {
		t.Fatalf("findForstFiles failed: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	expectedFiles := map[string]bool{
		"/test/file1.ft": false,
		"/test/file2.ft": false,
	}

	for _, file := range files {
		expectedFiles[file] = true
	}

	for file, found := range expectedFiles {
		if !found {
			t.Errorf("Expected file %s not found", file)
		}
	}
}

func TestDiscoverer_FindForstFiles_Error(t *testing.T) {
	logger := &MockLogger{}
	config := &MockConfig{
		err: fmt.Errorf("config error"),
	}

	discoverer := NewDiscoverer("/test/path", logger, config)

	_, err := discoverer.findForstFiles()
	if err == nil {
		t.Error("Expected error when config returns error")
	}
}

func TestDiscoverer_FindForstFiles_NilConfig(t *testing.T) {
	logger := &MockLogger{}

	discoverer := NewDiscoverer("/test/path", logger, nil)

	_, err := discoverer.findForstFiles()
	if err == nil {
		t.Error("Expected error when config is nil")
	}

	if !errors.Is(err, ErrNilForstConfig) {
		t.Errorf("Expected ErrNilForstConfig, got: %v", err)
	}
}

func TestDiscoverer_ExtractPackageNameFromAST(t *testing.T) {
	logger := &MockLogger{}
	discoverer := NewDiscoverer("/test/path", logger, nil)

	// Test with empty nodes
	result := discoverer.extractPackageNameFromAST([]ast.Node{})
	if result != "" {
		t.Errorf("Expected empty string for empty nodes, got '%s'", result)
	}

	// Test with non-package nodes
	result = discoverer.extractPackageNameFromAST([]ast.Node{
		&ast.FunctionNode{Ident: ast.Ident{ID: "TestFunc"}},
	})
	if result != "" {
		t.Errorf("Expected empty string for non-package nodes, got '%s'", result)
	}
}

func TestDiscoverer_ExtractFunctionsFromNode_PublicFunction(t *testing.T) {
	logger := &MockLogger{}
	discoverer := NewDiscoverer("/test/path", logger, nil)

	functions := make(map[string]FunctionInfo)

	// Create a public function node
	functionNode := &ast.FunctionNode{
		Ident: ast.Ident{ID: "TestFunction"},
		Params: []ast.ParamNode{
			&ast.SimpleParamNode{
				Ident: ast.Ident{ID: "input"},
				Type:  ast.TypeNode{Ident: "string"},
			},
		},
		ReturnTypes: []ast.TypeNode{
			{Ident: "string"},
		},
	}

	t.Logf("Created function node with ID: %s, type: %T", functionNode.Ident.ID, functionNode.Ident.ID)
	t.Logf("Function ID length: %d", len(functionNode.Ident.ID))
	if len(functionNode.Ident.ID) > 0 {
		t.Logf("First character: %c, is upper: %v", functionNode.Ident.ID[0], unicode.IsUpper(rune(functionNode.Ident.ID[0])))
	}

	discoverer.extractFunctionsFromNode(functionNode, "testpkg", "/test/file.ft", functions)

	if len(functions) != 1 {
		t.Errorf("Expected 1 function, got %d", len(functions))
	}

	fnInfo, exists := functions["TestFunction"]
	if !exists {
		t.Fatal("Function 'TestFunction' not found")
	}

	if fnInfo.Name != "TestFunction" {
		t.Errorf("Expected function name 'TestFunction', got '%s'", fnInfo.Name)
	}

	if fnInfo.Package != "testpkg" {
		t.Errorf("Expected package 'testpkg', got '%s'", fnInfo.Package)
	}

	if fnInfo.FilePath != "/test/file.ft" {
		t.Errorf("Expected file path '/test/file.ft', got '%s'", fnInfo.FilePath)
	}

	if len(fnInfo.Parameters) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(fnInfo.Parameters))
	}

	if fnInfo.Parameters[0].Name != "input" {
		t.Errorf("Expected parameter name 'input', got '%s'", fnInfo.Parameters[0].Name)
	}

	if fnInfo.ReturnType != "string" {
		t.Errorf("Expected return type 'string', got '%s'", fnInfo.ReturnType)
	}
}

func TestDiscoverer_ExtractFunctionsFromNode_PrivateFunction(t *testing.T) {
	logger := &MockLogger{}
	discoverer := NewDiscoverer("/test/path", logger, nil)

	functions := make(map[string]FunctionInfo)

	// Create a private function node
	functionNode := &ast.FunctionNode{
		Ident: ast.Ident{ID: "testFunction"}, // lowercase
		Params: []ast.ParamNode{
			&ast.SimpleParamNode{
				Ident: ast.Ident{ID: "input"},
				Type:  ast.TypeNode{Ident: "string"},
			},
		},
	}

	discoverer.extractFunctionsFromNode(functionNode, "testpkg", "/test/file.ft", functions)

	if len(functions) != 0 {
		t.Errorf("Expected 0 functions (private function), got %d", len(functions))
	}
}

func TestDiscoverer_ExtractFunctionsFromNode_NonFunctionNode(t *testing.T) {
	logger := &MockLogger{}
	discoverer := NewDiscoverer("/test/path", logger, nil)

	functions := make(map[string]FunctionInfo)

	// Create a non-function node
	packageNode := &ast.PackageNode{
		Ident: ast.Ident{ID: "testpkg"},
	}

	discoverer.extractFunctionsFromNode(packageNode, "testpkg", "/test/file.ft", functions)

	if len(functions) != 0 {
		t.Errorf("Expected 0 functions (non-function node), got %d", len(functions))
	}
}

func TestDiscoverer_AnalyzeStreamingSupport_FunctionName(t *testing.T) {
	logger := &MockLogger{}
	discoverer := NewDiscoverer("/test/path", logger, nil)

	tests := []struct {
		name     string
		expected bool
	}{
		{"StreamData", true},
		{"ProcessBatch", true},
		{"PipelineProcess", true},
		{"NormalFunction", false},
		{"GetUser", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			functionNode := &ast.FunctionNode{
				Ident: ast.Ident{ID: ast.Identifier(tt.name)},
			}

			result := discoverer.analyzeStreamingSupport(functionNode)
			if result != tt.expected {
				t.Errorf("Expected %v for function '%s', got %v", tt.expected, tt.name, result)
			}
		})
	}
}

func TestDiscoverer_AnalyzeStreamingSupport_ReturnType(t *testing.T) {
	logger := &MockLogger{}
	discoverer := NewDiscoverer("/test/path", logger, nil)

	tests := []struct {
		name     string
		expected bool
	}{
		{"Stream", true},
		{"Channel", true},
		{"string", false},
		{"int", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			functionNode := &ast.FunctionNode{
				Ident: ast.Ident{ID: "TestFunction"},
				ReturnTypes: []ast.TypeNode{
					{Ident: ast.TypeIdent(tt.name)},
				},
			}

			result := discoverer.analyzeStreamingSupport(functionNode)
			if result != tt.expected {
				t.Errorf("Expected %v for return type '%s', got %v", tt.expected, tt.name, result)
			}
		})
	}
}

func TestDiscoverer_TypeToString(t *testing.T) {
	logger := &MockLogger{}
	discoverer := NewDiscoverer("/test/path", logger, nil)

	tests := []struct {
		name     string
		expected string
	}{
		{"string", "string"},
		{"int", "int"},
		{"bool", "bool"},
		{"EchoRequest", "EchoRequest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeNode := ast.TypeNode{Ident: ast.TypeIdent(tt.name)}
			result := discoverer.typeToString(typeNode)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestDiscoverer_DetermineInputType(t *testing.T) {
	logger := &MockLogger{}
	discoverer := NewDiscoverer("/test/path", logger, nil)

	tests := []struct {
		name     string
		params   []ParameterInfo
		expected string
	}{
		{
			name:     "no parameters",
			params:   []ParameterInfo{},
			expected: "void",
		},
		{
			name: "single parameter",
			params: []ParameterInfo{
				{Name: "input", Type: "string"},
			},
			expected: "string",
		},
		{
			name: "multiple parameters",
			params: []ParameterInfo{
				{Name: "input", Type: "string"},
				{Name: "count", Type: "int"},
			},
			expected: "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := discoverer.determineInputType(tt.params)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestDiscoverer_ExtractFunctionsFromNodes(t *testing.T) {
	logger := &MockLogger{}
	discoverer := NewDiscoverer("/test/path", logger, nil)

	functions := make(map[string]FunctionInfo)

	nodes := []ast.Node{
		&ast.FunctionNode{
			Ident: ast.Ident{ID: "PublicFunction"},
			Params: []ast.ParamNode{
				&ast.SimpleParamNode{
					Ident: ast.Ident{ID: "input"},
					Type:  ast.TypeNode{Ident: "string"},
				},
			},
			ReturnTypes: []ast.TypeNode{
				{Ident: "string"},
			},
		},
		&ast.FunctionNode{
			Ident: ast.Ident{ID: "privateFunction"}, // lowercase
		},
		&ast.FunctionNode{
			Ident: ast.Ident{ID: "AnotherPublic"},
		},
	}

	discoverer.extractFunctionsFromNodes(nodes, "testpkg", "/test/file.ft", functions)

	if len(functions) != 2 {
		t.Errorf("Expected 2 public functions, got %d", len(functions))
	}

	// Check that only public functions are included
	expectedFunctions := map[string]bool{
		"PublicFunction":  false,
		"AnotherPublic":   false,
		"privateFunction": false, // should not be included
	}

	for name := range functions {
		expectedFunctions[name] = true
	}

	if expectedFunctions["privateFunction"] {
		t.Error("Private function should not be included")
	}

	if !expectedFunctions["PublicFunction"] {
		t.Error("PublicFunction should be included")
	}

	if !expectedFunctions["AnotherPublic"] {
		t.Error("AnotherPublic should be included")
	}
}

func TestDiscoverer_DiscoverFunctions_Integration(t *testing.T) {
	// Create a temporary test file
	tempDir, err := os.MkdirTemp("", "discovery-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.ft")
	testContent := `package testpkg

func PublicFunction(input String): String {
	return input
}

func privateFunction() {
	// private function
}`

	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create mock config that returns our test file
	config := &MockConfig{
		files: []string{testFile},
	}

	logger := &MockLogger{}
	discoverer := NewDiscoverer(tempDir, logger, config)

	functions, err := discoverer.DiscoverFunctions()
	if err != nil {
		t.Fatalf("DiscoverFunctions failed: %v", err)
	}

	// Check that we found the testpkg package
	pkgFuncs, exists := functions["testpkg"]
	if !exists {
		t.Fatal("Package 'testpkg' not found")
	}

	// Check that we found the public functions
	if len(pkgFuncs) != 1 {
		t.Errorf("Expected 1 public function, got %d", len(pkgFuncs))
	}

	// Check PublicFunction
	pubFunc, exists := pkgFuncs["PublicFunction"]
	if !exists {
		t.Error("PublicFunction not found")
	} else {
		if pubFunc.Name != "PublicFunction" {
			t.Errorf("Expected function name 'PublicFunction', got '%s'", pubFunc.Name)
		}
		if pubFunc.Package != "testpkg" {
			t.Errorf("Expected package 'testpkg', got '%s'", pubFunc.Package)
		}
		if len(pubFunc.Parameters) != 1 {
			t.Errorf("Expected 1 parameter, got %d", len(pubFunc.Parameters))
		}
		if pubFunc.Parameters[0].Name != "input" {
			t.Errorf("Expected parameter name 'input', got '%s'", pubFunc.Parameters[0].Name)
		}
		if pubFunc.Parameters[0].Type != "String" {
			t.Errorf("Expected parameter type 'String', got '%s'", pubFunc.Parameters[0].Type)
		}
		if pubFunc.ReturnType != "String" {
			t.Errorf("Expected return type 'String', got '%s'", pubFunc.ReturnType)
		}
		if pubFunc.SupportsStreaming {
			t.Error("PublicFunction should not support streaming")
		}
	}
}

func TestDiscoverer_DiscoverFunctions_NoFiles(t *testing.T) {
	config := &MockConfig{
		files: []string{},
	}

	logger := &MockLogger{}
	discoverer := NewDiscoverer("/test/path", logger, config)

	functions, err := discoverer.DiscoverFunctions()
	if err != nil {
		t.Fatalf("DiscoverFunctions failed: %v", err)
	}

	if len(functions) != 0 {
		t.Errorf("Expected 0 packages, got %d", len(functions))
	}
}

func TestDiscoverer_DiscoverFunctions_ConfigError(t *testing.T) {
	config := &MockConfig{
		err: fmt.Errorf("config error"),
	}

	logger := &MockLogger{}
	discoverer := NewDiscoverer("/test/path", logger, config)

	_, err := discoverer.DiscoverFunctions()
	if err == nil {
		t.Error("Expected error when config fails")
	}
}

func TestDiscoverer_FunctionInfo_JSON(t *testing.T) {
	// Test that FunctionInfo can be marshaled to JSON
	fnInfo := FunctionInfo{
		Package:           "testpkg",
		Name:              "TestFunction",
		SupportsStreaming: true,
		InputType:         "string",
		OutputType:        "string",
		Parameters: []ParameterInfo{
			{Name: "input", Type: "string"},
		},
		ReturnType: "string",
		FilePath:   "/test/file.ft",
	}

	_, err := json.Marshal(fnInfo)
	if err != nil {
		t.Errorf("Failed to marshal FunctionInfo to JSON: %v", err)
	}
}

func TestDiscoverer_ParameterInfo_JSON(t *testing.T) {
	// Test that ParameterInfo can be marshaled to JSON
	paramInfo := ParameterInfo{
		Name: "input",
		Type: "string",
	}

	_, err := json.Marshal(paramInfo)
	if err != nil {
		t.Errorf("Failed to marshal ParameterInfo to JSON: %v", err)
	}
}
