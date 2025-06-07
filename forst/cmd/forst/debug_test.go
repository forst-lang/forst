package main

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/typechecker"
	"runtime"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

// TestHook is a custom hook to capture log entries
type TestHook struct {
	entries []*logrus.Entry
}

func (h *TestHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *TestHook) Fire(entry *logrus.Entry) error {
	h.entries = append(h.entries, entry)
	return nil
}

func (h *TestHook) Reset() {
	h.entries = nil
}

func (h *TestHook) ContainsMessage(msg string) bool {
	for _, entry := range h.entries {
		// Check the message field
		if strings.Contains(entry.Message, msg) {
			return true
		}
		// Check all fields
		for k, v := range entry.Data {
			if strings.Contains(k, msg) || strings.Contains(fmt.Sprint(v), msg) {
				return true
			}
		}
	}
	return false
}

var testHook = &TestHook{}

func init() {
	// Set log level to debug for testing
	logrus.SetLevel(logrus.DebugLevel)
	// Add our test hook
	logrus.AddHook(testHook)
}

func TestLogMemUsage(t *testing.T) {
	testHook.Reset()

	// Create a program with memory reporting enabled
	program := NewProgram(ProgramArgs{
		reportMemoryUsage: true,
	})

	// Get initial memory stats
	before := runtime.MemStats{}
	runtime.ReadMemStats(&before)

	// Allocate some memory
	_ = make([]byte, 1024*1024) // 1MB allocation

	// Get memory stats after allocation
	after := runtime.MemStats{}
	runtime.ReadMemStats(&after)

	// Test memory usage logging
	program.logMemUsage("test_phase", before, after)

	// Verify that memory usage was logged
	if !testHook.ContainsMessage("test_phase") {
		t.Error("Expected memory usage log to contain 'test_phase'")
	}
	if !testHook.ContainsMessage("allocatedBytes") {
		t.Error("Expected memory usage log to contain 'allocatedBytes'")
	}
}

func TestDebugPrintTokens(t *testing.T) {
	testHook.Reset()

	// Create a program with debug enabled
	program := NewProgram(ProgramArgs{
		debug: true,
	})

	// Create test tokens
	tokens := []ast.Token{
		{
			Type:   ast.TokenIdentifier,
			Value:  "test",
			Path:   "test.ft",
			Line:   1,
			Column: 1,
		},
		{
			Type:   ast.TokenIntLiteral,
			Value:  "42",
			Path:   "test.ft",
			Line:   1,
			Column: 6,
		},
	}

	// Test token printing
	program.debugPrintTokens(tokens)

	// Verify that tokens were logged
	if !testHook.ContainsMessage("IDENTIFIER") {
		t.Error("Expected token log to contain 'IDENTIFIER'")
	}
	if !testHook.ContainsMessage("INT_LITERAL") {
		t.Error("Expected token log to contain 'INT_LITERAL'")
	}
}

func TestDebugPrintForstAST(t *testing.T) {
	testHook.Reset()

	// Create a program with debug enabled
	program := NewProgram(ProgramArgs{
		debug: true,
	})

	// Create test AST nodes
	nodes := []ast.Node{
		ast.PackageNode{Ident: ast.Ident{ID: "main"}},
		ast.ImportNode{Path: "fmt"},
		ast.FunctionNode{
			Ident:       ast.Ident{ID: "main"},
			ReturnTypes: []ast.TypeNode{{Ident: ast.TypeVoid}},
			Body:        []ast.Node{},
		},
	}

	// Test AST printing
	program.debugPrintForstAST(nodes)

	// Verify that AST nodes were logged
	if !testHook.ContainsMessage("Package declaration") {
		t.Error("Expected AST log to contain 'Package declaration'")
	}
	if !testHook.ContainsMessage("Import") {
		t.Error("Expected AST log to contain 'Import'")
	}
}

func TestDebugPrintTypeInfo(t *testing.T) {
	testHook.Reset()

	// Create a program with debug enabled
	program := NewProgram(ProgramArgs{
		debug: true,
	})

	// Create a type checker with some test data
	tc := typechecker.New()
	tc.Functions["main"] = typechecker.FunctionSignature{
		Ident: ast.Ident{ID: "main"},
		Parameters: []typechecker.ParameterSignature{
			{
				Ident: ast.Ident{ID: "x"},
				Type:  ast.TypeNode{Ident: ast.TypeInt},
			},
		},
		ReturnTypes: []ast.TypeNode{
			{Ident: ast.TypeVoid},
		},
	}

	tc.Defs["x"] = ast.TypeDefNode{
		Ident: ast.TypeIdent("x"),
		Expr:  ast.TypeDefAssertionExpr{},
	}

	// Test type info printing
	program.debugPrintTypeInfo(tc)

	// Verify that type info was logged
	if !testHook.ContainsMessage("function signature") {
		t.Error("Expected type info log to contain 'function signature'")
	}
	if !testHook.ContainsMessage("definition") {
		t.Error("Expected type info log to contain 'definition'")
	}
}
