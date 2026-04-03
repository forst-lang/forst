package lexer

import (
	"io"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func testLogger(t *testing.T) *logrus.Logger {
	t.Helper()
	log := logrus.New()
	log.SetOutput(io.Discard)
	return log
}

func TestLex_packageLine(t *testing.T) {
	l := New([]byte("package main\n"), "p.ft", testLogger(t))
	toks := l.Lex()
	if len(toks) < 3 {
		t.Fatalf("expected at least 3 tokens, got %d: %+v", len(toks), toks)
	}
	if toks[0].Type != ast.TokenPackage || toks[0].Value != "package" {
		t.Fatalf("first token: %+v", toks[0])
	}
	if toks[1].Type != ast.TokenIdentifier || toks[1].Value != "main" {
		t.Fatalf("second token: %+v", toks[1])
	}
	if toks[len(toks)-1].Type != ast.TokenEOF {
		t.Fatalf("last token should be EOF, got %+v", toks[len(toks)-1])
	}
}

func TestLex_operatorsAndLiterals(t *testing.T) {
	l := New([]byte(`x := 1`), "f.ft", testLogger(t))
	toks := l.Lex()
	// x, :=, 1, EOF
	if len(toks) < 4 {
		t.Fatalf("tokens: %+v", toks)
	}
	if toks[0].Value != "x" || toks[0].Type != ast.TokenIdentifier {
		t.Fatalf("ident: %+v", toks[0])
	}
	if toks[1].Type != ast.TokenColonEquals {
		t.Fatalf(":= : %+v", toks[1])
	}
	if toks[2].Type != ast.TokenIntLiteral || toks[2].Value != "1" {
		t.Fatalf("literal: %+v", toks[2])
	}
}
