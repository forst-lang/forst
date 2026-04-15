package discovery

import (
	"testing"
	"unicode"

	"forst/internal/ast"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func TestDiscoverer_ExtractPackageNameFromAST(t *testing.T) {
	logger := &MockLogger{}
	discoverer := NewDiscoverer("/test/path", logger, nil)

	result := discoverer.extractPackageNameFromAST([]ast.Node{})
	if result != "" {
		t.Errorf("Expected empty string for empty nodes, got '%s'", result)
	}

	result = discoverer.extractPackageNameFromAST([]ast.Node{
		&ast.FunctionNode{Ident: ast.Ident{ID: "TestFunc"}},
	})
	if result != "" {
		t.Errorf("Expected empty string for non-package nodes, got '%s'", result)
	}

	result = discoverer.extractPackageNameFromAST([]ast.Node{
		ast.PackageNode{Ident: ast.Ident{ID: "pkgname"}},
	})
	if result != "pkgname" {
		t.Errorf("Expected package name pkgname, got %q", result)
	}
}

func TestDiscoverer_ExtractFunctionsFromNode_PublicFunction(t *testing.T) {
	logger := &MockLogger{}
	discoverer := NewDiscoverer("/test/path", logger, nil)
	functions := make(map[string]FunctionInfo)

	functionNode := &ast.FunctionNode{
		Ident: ast.Ident{ID: "TestFunction"},
		Params: []ast.ParamNode{
			&ast.SimpleParamNode{
				Ident: ast.Ident{ID: "input"},
				Type:  ast.TypeNode{Ident: "string"},
			},
		},
		ReturnTypes: []ast.TypeNode{{Ident: "string"}},
	}

	t.Logf("Created function node with ID: %s, type: %T", functionNode.Ident.ID, functionNode.Ident.ID)
	t.Logf("Function ID length: %d", len(functionNode.Ident.ID))
	if len(functionNode.Ident.ID) > 0 {
		t.Logf("First character: %c, is upper: %v", functionNode.Ident.ID[0], unicode.IsUpper(rune(functionNode.Ident.ID[0])))
	}

	discoverer.extractFunctionsFromNode(functionNode, "testpkg", "/test/file.ft", functions, nil)

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
	discoverer := NewDiscoverer("/test/path", &MockLogger{}, nil)
	functions := make(map[string]FunctionInfo)
	functionNode := &ast.FunctionNode{
		Ident: ast.Ident{ID: "testFunction"},
		Params: []ast.ParamNode{
			&ast.SimpleParamNode{
				Ident: ast.Ident{ID: "input"},
				Type:  ast.TypeNode{Ident: "string"},
			},
		},
	}

	discoverer.extractFunctionsFromNode(functionNode, "testpkg", "/test/file.ft", functions, nil)
	if len(functions) != 0 {
		t.Errorf("Expected 0 functions (private function), got %d", len(functions))
	}
}

func TestDiscoverer_ExtractFunctionsFromNode_NonFunctionNode(t *testing.T) {
	discoverer := NewDiscoverer("/test/path", &MockLogger{}, nil)
	functions := make(map[string]FunctionInfo)
	packageNode := &ast.PackageNode{Ident: ast.Ident{ID: "testpkg"}}
	discoverer.extractFunctionsFromNode(packageNode, "testpkg", "/test/file.ft", functions, nil)
	if len(functions) != 0 {
		t.Errorf("Expected 0 functions (non-function node), got %d", len(functions))
	}
}

func TestDiscoverer_ExtractFunctionsFromNodes(t *testing.T) {
	logger := logrus.New()
	discoverer := NewDiscoverer("/test/path", logger, nil)
	tc := typechecker.New(logger, false)
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
			ReturnTypes: []ast.TypeNode{{Ident: "string"}},
		},
		&ast.FunctionNode{Ident: ast.Ident{ID: "privateFunction"}},
		&ast.FunctionNode{Ident: ast.Ident{ID: "AnotherPublic"}},
	}

	discoverer.extractFunctionsFromNodes(nodes, "testpkg", "/test/file.ft", functions, tc)
	if len(functions) != 2 {
		t.Errorf("Expected 2 public functions, got %d", len(functions))
	}
	if _, ok := functions["privateFunction"]; ok {
		t.Error("Private function should not be included")
	}
	if _, ok := functions["PublicFunction"]; !ok {
		t.Error("PublicFunction should be included")
	}
	if _, ok := functions["AnotherPublic"]; !ok {
		t.Error("AnotherPublic should be included")
	}
}

func TestDiscoverer_extractFunctionsFromNode_FunctionNodeValueReceiver(t *testing.T) {
	discoverer := NewDiscoverer("/test/path", &MockLogger{}, nil)
	functions := make(map[string]FunctionInfo)
	node := ast.FunctionNode{
		Ident:       ast.Ident{ID: "Exported"},
		ReturnTypes: []ast.TypeNode{{Ident: "String"}},
	}
	var n ast.Node = node
	discoverer.extractFunctionsFromNode(n, "pkg", "/f.ft", functions, nil)
	if len(functions) != 1 || functions["Exported"].Name != "Exported" {
		t.Fatalf("expected Exported function, got %+v", functions)
	}
}
