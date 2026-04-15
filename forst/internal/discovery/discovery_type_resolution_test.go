package discovery

import (
	"encoding/json"
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func TestDiscoverer_TypeToString(t *testing.T) {
	discoverer := NewDiscoverer("/test/path", &MockLogger{}, nil)
	tests := []struct {
		name     string
		expected string
	}{
		{"string", "string"},
		{"int", "int"},
		{"bool", "bool"},
		{"EchoRequest", "EchoRequest"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			typeNode := ast.TypeNode{Ident: ast.TypeIdent(testCase.name)}
			result := discoverer.typeToString(typeNode)
			if result != testCase.expected {
				t.Errorf("Expected '%s', got '%s'", testCase.expected, result)
			}
		})
	}
}

func TestDiscoverer_DetermineInputType(t *testing.T) {
	discoverer := NewDiscoverer("/test/path", &MockLogger{}, nil)
	tests := []struct {
		name     string
		params   []ParameterInfo
		expected string
	}{
		{name: "no parameters", params: []ParameterInfo{}, expected: "void"},
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

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := discoverer.determineInputType(testCase.params)
			if result != testCase.expected {
				t.Errorf("Expected '%s', got '%s'", testCase.expected, result)
			}
		})
	}
}

func TestDiscoverer_applyResultDiscoveryMetadata_nonResultNoChange(t *testing.T) {
	discoverer := NewDiscoverer("/", logrus.New(), nil)
	functionInfo := FunctionInfo{
		Name:               "Plain",
		IsResult:           false,
		ResultSuccessType:  "",
		ResultFailureType:  "",
		HasMultipleReturns: false,
	}

	discoverer.applyResultDiscoveryMetadata(&functionInfo, ast.NewBuiltinType(ast.TypeString), nil)
	if functionInfo.IsResult || functionInfo.ResultSuccessType != "" || functionInfo.ResultFailureType != "" || functionInfo.HasMultipleReturns {
		t.Fatalf("expected unchanged FunctionInfo for non-result type, got %+v", functionInfo)
	}
}

func TestDiscoveryLogrusOrDiscard_returnsLoggerForBothInputs(t *testing.T) {
	stdLogger := logrus.New()
	gotStd := discoveryLogrusOrDiscard(stdLogger)
	if gotStd != stdLogger {
		t.Fatal("expected original logrus logger to be reused")
	}

	mockLogger := &MockLogger{}
	gotFallback := discoveryLogrusOrDiscard(mockLogger)
	if gotFallback == nil {
		t.Fatal("expected fallback logger for non-logrus logger")
	}
	if gotFallback == stdLogger {
		t.Fatal("expected distinct fallback logger for non-logrus logger")
	}
}

func TestDiscoverer_resolveFunctionReturnTypes_prefersTypecheckerWhenAvailable(t *testing.T) {
	logger := logrus.New()
	discoverer := NewDiscoverer("/", logger, nil)
	typeChecker := typechecker.New(logger, false)
	typeChecker.Functions[ast.Identifier("F")] = typechecker.FunctionSignature{
		Ident:       ast.Ident{ID: ast.Identifier("F")},
		ReturnTypes: []ast.TypeNode{ast.NewBuiltinType(ast.TypeInt)},
	}

	functionNode := &ast.FunctionNode{
		Ident:       ast.Ident{ID: ast.Identifier("F")},
		ReturnTypes: []ast.TypeNode{ast.NewBuiltinType(ast.TypeString)},
	}
	got := discoverer.resolveFunctionReturnTypes(functionNode, typeChecker)
	if len(got) != 1 || got[0].Ident != ast.TypeInt {
		t.Fatalf("expected inferred typechecker return Int, got %+v", got)
	}
}

func TestDiscoverer_resolveFunctionReturnTypes_fallsBackToParser(t *testing.T) {
	logger := logrus.New()
	discoverer := NewDiscoverer("/", logger, nil)
	typeChecker := typechecker.New(logger, false)

	functionNode := &ast.FunctionNode{
		Ident:       ast.Ident{ID: ast.Identifier("NoSig")},
		ReturnTypes: []ast.TypeNode{ast.NewBuiltinType(ast.TypeString)},
	}
	got := discoverer.resolveFunctionReturnTypes(functionNode, typeChecker)
	if len(got) != 1 || got[0].Ident != ast.TypeString {
		t.Fatalf("expected parser return String fallback, got %+v", got)
	}
}

func TestDiscoverer_FunctionInfo_JSON(t *testing.T) {
	functionInfo := FunctionInfo{
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

	if _, err := json.Marshal(functionInfo); err != nil {
		t.Errorf("Failed to marshal FunctionInfo to JSON: %v", err)
	}
}

func TestDiscoverer_ParameterInfo_JSON(t *testing.T) {
	parameterInfo := ParameterInfo{Name: "input", Type: "string"}
	if _, err := json.Marshal(parameterInfo); err != nil {
		t.Errorf("Failed to marshal ParameterInfo to JSON: %v", err)
	}
}
