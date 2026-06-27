package lexer

import (
	"forst/internal/ast"
	"testing"
)

func TestLexer_useAndWithKeywords(t *testing.T) {
	input := "use with"
	expected := []ast.Token{
		{Type: ast.TokenUse, Value: "use", FileID: testFileID, Line: 1, Column: 1},
		{Type: ast.TokenWith, Value: "with", FileID: testFileID, Line: 1, Column: 5},
		{Type: ast.TokenEOF, Value: "", FileID: testFileID, Line: 2, Column: 1},
	}
	l := New([]byte(input), testFileID, nil)
	got := l.Lex()
	if len(got) != len(expected) {
		t.Fatalf("token count: got %d want %d", len(got), len(expected))
	}
	for i := range expected {
		if got[i].Type != expected[i].Type || got[i].Value != expected[i].Value {
			t.Fatalf("token[%d]: got %s %q want %s %q", i, got[i].Type, got[i].Value, expected[i].Type, expected[i].Value)
		}
	}
}
