package discovery

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func TestDiscoverer_AnalyzeStreamingSupport_FunctionName(t *testing.T) {
	logger := logrus.New()
	tc := typechecker.New(logger, false)

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

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			functionNode := &ast.FunctionNode{
				Ident: ast.Ident{ID: ast.Identifier(testCase.name)},
			}
			result := StreamingSupported(functionNode, tc)
			if result != testCase.expected {
				t.Errorf("Expected %v for function '%s', got %v", testCase.expected, testCase.name, result)
			}
		})
	}
}

func TestDiscoverer_AnalyzeStreamingSupport_ReturnType(t *testing.T) {
	logger := logrus.New()
	tc := typechecker.New(logger, false)

	tests := []struct {
		name     string
		expected bool
	}{
		{"Stream", true},
		{"Channel", true},
		{"string", false},
		{"int", false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			functionNode := &ast.FunctionNode{
				Ident: ast.Ident{ID: "TestFunction"},
				ReturnTypes: []ast.TypeNode{
					ast.NewUserDefinedType(ast.TypeIdent(testCase.name)),
				},
			}
			result := StreamingSupported(functionNode, tc)
			if result != testCase.expected {
				t.Errorf("Expected %v for return type '%s', got %v", testCase.expected, testCase.name, result)
			}
		})
	}
}

func TestDiscoverer_AnalyzeStreamingSupport_nilTypecheckerUsesReturnTypes(t *testing.T) {
	functionNode := &ast.FunctionNode{
		Ident: ast.Ident{ID: "Plain"},
		ReturnTypes: []ast.TypeNode{
			{Ident: "UserStream"},
		},
	}
	if !StreamingSupported(functionNode, nil) {
		t.Fatal("expected stream hint from return type name when tc is nil")
	}
}

func TestStreamingSupported_channelReturn(t *testing.T) {
	tc := typechecker.New(logrus.New(), false)
	fn := &ast.FunctionNode{
		Ident: ast.Ident{ID: "PlainName"},
		ReturnTypes: []ast.TypeNode{
			ast.NewChannelType(ast.NewBuiltinType(ast.TypeString)),
		},
	}
	if !StreamingSupported(fn, tc) {
		t.Fatal("expected streaming for chan return")
	}
}
