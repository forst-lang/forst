package compiler

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/typechecker"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
)

// TestHook is a custom hook to capture log entries
type TestHook struct {
	entries []*logrus.Entry
	mu      sync.Mutex
}

func (h *TestHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *TestHook) Fire(entry *logrus.Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = append(h.entries, entry)
	return nil
}

func (h *TestHook) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = nil
}

func (h *TestHook) ContainsMessage(msg string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
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

func TestLogMemUsage(t *testing.T) {
	hook := &TestHook{}

	log := setupTestLogger(nil)
	log.AddHook(hook)

	// Create a compiler with memory reporting enabled
	c := New(Args{
		ReportMemoryUsage: true,
	}, log)

	// Get initial memory stats
	before := runtime.MemStats{}
	runtime.ReadMemStats(&before)

	// Allocate some memory
	_ = make([]byte, 1024*1024) // 1MB allocation

	// Get memory stats after allocation
	after := runtime.MemStats{}
	runtime.ReadMemStats(&after)

	// Test memory usage logging
	c.logMemUsage("test_phase", before, after)

	// Verify that memory usage was logged
	if !hook.ContainsMessage("test_phase") {
		t.Error("Expected memory usage log to contain 'test_phase'")
	}
	if !hook.ContainsMessage("allocatedBytes") {
		t.Error("Expected memory usage log to contain 'allocatedBytes'")
	}
}

func TestDebugPrintTokens(t *testing.T) {
	hook := &TestHook{}

	log := setupTestLogger(&ast.TestLoggerOptions{ForceLevel: logrus.DebugLevel})
	log.AddHook(hook)

	// Create a compiler with debug enabled
	c := New(Args{
		LogLevel: "debug",
	}, log)

	// Create test tokens
	tokens := []ast.Token{
		{
			Type:   ast.TokenIdentifier,
			Value:  "test",
			FileID: "test.ft",
			Line:   1,
			Column: 1,
		},
		{
			Type:   ast.TokenIntLiteral,
			Value:  "42",
			FileID: "test.ft",
			Line:   1,
			Column: 6,
		},
	}

	// Test token printing
	c.debugPrintTokens(tokens)

	// Verify that tokens were logged
	// Debug: let's see what was actually captured
	t.Logf("Hook captured %d entries", len(hook.entries))
	for i, entry := range hook.entries {
		t.Logf("Entry %d: %s", i, entry.Message)
	}

	if !hook.ContainsMessage("IDENTIFIER") {
		t.Error("Expected token log to contain 'IDENTIFIER'")
	}
	if !hook.ContainsMessage("INT_LITERAL") {
		t.Error("Expected token log to contain 'INT_LITERAL'")
	}
}

func TestDebugPrintForstAST(t *testing.T) {
	hook := &TestHook{}

	log := setupTestLogger(&ast.TestLoggerOptions{ForceLevel: logrus.DebugLevel})
	log.AddHook(hook)

	// Create a compiler with debug enabled
	c := New(Args{
		LogLevel: "debug",
	}, log)

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
	c.debugPrintForstAST(nodes)

	// Verify that AST nodes were logged
	if !hook.ContainsMessage("Package declaration") {
		t.Error("Expected AST log to contain 'Package declaration'")
	}
	if !hook.ContainsMessage("Import") {
		t.Error("Expected AST log to contain 'Import'")
	}
}

func TestDebugPrintTypeInfo(t *testing.T) {
	hook := &TestHook{}

	log := setupTestLogger(&ast.TestLoggerOptions{ForceLevel: logrus.DebugLevel})
	log.AddHook(hook)

	// Create a compiler with debug enabled
	c := New(Args{
		LogLevel: "debug",
	}, log)

	// Create a type checker with some test data
	tc := typechecker.New(log, false)
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
	c.debugPrintTypeInfo(tc)

	// Verify that type info was logged
	if !hook.ContainsMessage("function signature") {
		t.Error("Expected type info log to contain 'function signature'")
	}
	if !hook.ContainsMessage("definition") {
		t.Error("Expected type info log to contain 'definition'")
	}
}
