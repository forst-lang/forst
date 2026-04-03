package lexer

import (
	"testing"

	"forst/internal/ast"
)

func TestProcessStringLiteral_doubleQuoted(t *testing.T) {
	line := []byte(`"hello"`)
	tok, next := processStringLiteral(line, 0, "f.ft", 1)
	if next != len(line) {
		t.Fatalf("next col: got %d want %d", next, len(line))
	}
	if tok.Type != ast.TokenStringLiteral || tok.Value != `"hello"` {
		t.Fatalf("token: %+v", tok)
	}
	if tok.Line != 1 || tok.Column != 1 || tok.FileID != "f.ft" {
		t.Fatalf("position: %+v", tok)
	}
}

func TestProcessStringLiteral_backtick(t *testing.T) {
	line := []byte("`a\nb`")
	tok, next := processStringLiteral(line, 0, "f.ft", 2)
	if tok.Type != ast.TokenStringLiteral || tok.Value != "`a\nb`" {
		t.Fatalf("backtick token: %+v", tok)
	}
	if next != len(line) {
		t.Fatalf("next: %d", next)
	}
}

func TestProcessSpecialChar_lineComment(t *testing.T) {
	line := []byte("// comment")
	tok, next := processSpecialChar(line, 0, "f.ft", 1)
	if tok.Type != ast.TokenComment {
		t.Fatalf("want comment, got %+v", tok)
	}
	if tok.Value != "// comment" || next != len(line) {
		t.Fatalf("comment token: %+v next=%d", tok, next)
	}
}

func TestProcessSpecialChar_twoCharOperator(t *testing.T) {
	line := []byte("==x")
	tok, next := processSpecialChar(line, 0, "f.ft", 1)
	if tok.Type != ast.TokenEquals || tok.Value != "==" {
		t.Fatalf("token: %+v", tok)
	}
	if next != 2 {
		t.Fatalf("next: %d", next)
	}
}

func TestProcessSpecialChar_incrementDecrement(t *testing.T) {
	for _, tt := range []struct {
		line string
		want ast.TokenIdent
	}{
		{"++x", ast.TokenPlusPlus},
		{"--y", ast.TokenMinusMinus},
	} {
		tok, next := processSpecialChar([]byte(tt.line), 0, "f.ft", 1)
		if tok.Type != tt.want || len(tok.Value) != 2 {
			t.Fatalf("%q: got %+v", tt.line, tok)
		}
		if next != 2 {
			t.Fatalf("%q: next %d", tt.line, next)
		}
	}
}

func TestProcessSpecialChar_semicolon(t *testing.T) {
	line := []byte(";x")
	tok, next := processSpecialChar(line, 0, "f.ft", 1)
	if tok.Type != ast.TokenSemicolon || tok.Value != ";" {
		t.Fatalf("token: %+v", tok)
	}
	if next != 1 {
		t.Fatalf("next: %d", next)
	}
}

func TestProcessWord_identifier(t *testing.T) {
	line := []byte("foo ")
	tok, next := processWord(line, 0, "f.ft", 1)
	if tok.Type != ast.TokenIdentifier || tok.Value != "foo" {
		t.Fatalf("token: %+v", tok)
	}
	if next != 3 {
		t.Fatalf("next: %d", next)
	}
}

func TestProcessWord_identifier_stopsBeforeSemicolon(t *testing.T) {
	line := []byte("n;n")
	tok, next := processWord(line, 0, "f.ft", 1)
	if tok.Type != ast.TokenIdentifier || tok.Value != "n" {
		t.Fatalf("token: %+v", tok)
	}
	if next != 1 {
		t.Fatalf("next: %d", next)
	}
}

func TestProcessWord_intLiteral(t *testing.T) {
	line := []byte("42)")
	tok, next := processWord(line, 0, "f.ft", 1)
	if tok.Type != ast.TokenIntLiteral || tok.Value != "42" {
		t.Fatalf("token: %+v", tok)
	}
	if next != 2 {
		t.Fatalf("next: %d", next)
	}
}
