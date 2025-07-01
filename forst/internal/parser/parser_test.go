package parser

import (
	"forst/internal/ast"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestNewParser(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenEOF, Value: "", Line: 1, Column: 1},
	}

	p := New(tokens, "test.ft", nil)

	if p == nil {
		t.Fatal("Expected parser to be created, got nil")
	}

	if len(p.tokens) != 1 {
		t.Errorf("Expected 1 token, got %d", len(p.tokens))
	}

	if p.currentIndex != 0 {
		t.Errorf("Expected currentIndex to be 0, got %d", p.currentIndex)
	}

	if p.context == nil {
		t.Fatal("Expected context to be created, got nil")
	}

	if p.context.FilePath != "test.ft" {
		t.Errorf("Expected file path 'test.ft', got %s", p.context.FilePath)
	}

	if p.context.ScopeStack == nil {
		t.Fatal("Expected scope stack to be created, got nil")
	}

	if p.log == nil {
		t.Fatal("Expected logger to be created, got nil")
	}
}

func TestParserCurrent(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenIdentifier, Value: "x", Line: 1, Column: 1},
		{Type: ast.TokenEOF, Value: "", Line: 1, Column: 2},
	}

	p := New(tokens, "test.ft", nil)

	// Test current token
	current := p.current()
	if current.Type != ast.TokenIdentifier {
		t.Errorf("Expected current token type TokenIdentifier, got %s", current.Type)
	}
	if current.Value != "x" {
		t.Errorf("Expected current token value 'x', got %s", current.Value)
	}

	// Test current token at end
	p.currentIndex = 1
	current = p.current()
	if current.Type != ast.TokenEOF {
		t.Errorf("Expected current token type TokenEOF, got %s", current.Type)
	}

	// Test current token beyond end
	p.currentIndex = 2
	current = p.current()
	if current.Type != ast.TokenEOF {
		t.Errorf("Expected current token type TokenEOF, got %s", current.Type)
	}
}

func TestParserPeek(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenIdentifier, Value: "x", Line: 1, Column: 1},
		{Type: ast.TokenEquals, Value: "=", Line: 1, Column: 3},
		{Type: ast.TokenIntLiteral, Value: "42", Line: 1, Column: 5},
		{Type: ast.TokenEOF, Value: "", Line: 1, Column: 7},
	}

	p := New(tokens, "test.ft", nil)

	// Test peek with default offset (1)
	peeked := p.peek()
	if peeked.Type != ast.TokenEquals {
		t.Errorf("Expected peeked token type TokenEquals, got %s", peeked.Type)
	}

	// Test peek with specific offset
	peeked = p.peek(2)
	if peeked.Type != ast.TokenIntLiteral {
		t.Errorf("Expected peeked token type TokenIntLiteral, got %s", peeked.Type)
	}

	// Test peek beyond end
	peeked = p.peek(10)
	if peeked.Type != ast.TokenEOF {
		t.Errorf("Expected peeked token type TokenEOF, got %s", peeked.Type)
	}
}

func TestParserAdvance(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenIdentifier, Value: "x", Line: 1, Column: 1},
		{Type: ast.TokenEquals, Value: "=", Line: 1, Column: 3},
		{Type: ast.TokenEOF, Value: "", Line: 1, Column: 5},
	}

	p := New(tokens, "test.ft", nil)

	// Test initial position
	if p.currentIndex != 0 {
		t.Errorf("Expected initial currentIndex to be 0, got %d", p.currentIndex)
	}

	// Test advance
	advanced := p.advance()
	if p.currentIndex != 1 {
		t.Errorf("Expected currentIndex to be 1 after advance, got %d", p.currentIndex)
	}
	if advanced.Type != ast.TokenEquals {
		t.Errorf("Expected advanced token type TokenEquals, got %s", advanced.Type)
	}

	// Test advance again
	advanced = p.advance()
	if p.currentIndex != 2 {
		t.Errorf("Expected currentIndex to be 2 after second advance, got %d", p.currentIndex)
	}
	if advanced.Type != ast.TokenEOF {
		t.Errorf("Expected advanced token type TokenEOF, got %s", advanced.Type)
	}

	// Test advance at end
	advanced = p.advance()
	if p.currentIndex != 3 {
		t.Errorf("Expected currentIndex to be 3 after advance at end, got %d", p.currentIndex)
	}
	if advanced.Type != ast.TokenEOF {
		t.Errorf("Expected advanced token type TokenEOF, got %s", advanced.Type)
	}
}

func TestParserExpect(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenIdentifier, Value: "x", Line: 1, Column: 1},
		{Type: ast.TokenEOF, Value: "", Line: 1, Column: 2},
	}

	p := New(tokens, "test.ft", nil)

	// Test successful expect
	expected := p.expect(ast.TokenIdentifier)
	if expected.Type != ast.TokenIdentifier {
		t.Errorf("Expected token type TokenIdentifier, got %s", expected.Type)
	}
	if expected.Value != "x" {
		t.Errorf("Expected token value 'x', got %s", expected.Value)
	}
	if p.currentIndex != 1 {
		t.Errorf("Expected currentIndex to be 1 after expect, got %d", p.currentIndex)
	}

	// Test expect with wrong token type
	p = New(tokens, "test.ft", nil)
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for wrong token type, but no panic occurred")
		}
	}()
	p.expect(ast.TokenEquals) // This should panic
}

func TestParserContext(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenEOF, Value: "", Line: 1, Column: 1},
	}

	p := New(tokens, "test.ft", nil)

	// Test context initialization
	if p.context == nil {
		t.Fatal("Expected context to be initialized, got nil")
	}

	if p.context.FilePath != "test.ft" {
		t.Errorf("Expected file path 'test.ft', got %s", p.context.FilePath)
	}

	if p.context.Package != nil {
		t.Error("Expected package to be nil initially")
	}

	if p.context.Function != nil {
		t.Error("Expected function to be nil initially")
	}

	if p.context.ScopeStack == nil {
		t.Fatal("Expected scope stack to be initialized, got nil")
	}
}

func TestParserWithCustomLogger(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenEOF, Value: "", Line: 1, Column: 1},
	}

	logger := logrus.New()
	p := New(tokens, "test.ft", logger)

	if p.log != logger {
		t.Error("Expected parser to use provided logger")
	}
}
