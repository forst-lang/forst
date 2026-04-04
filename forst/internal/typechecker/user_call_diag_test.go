package typechecker

import (
	"errors"
	"testing"

	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestUserFunctionCall_wrongArityDiagnostic(t *testing.T) {
	src := "package main\nfunc f(a Int) {}\nfunc main() {\n  f(1, 2)\n}\n"
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected error")
	}
	var diag *Diagnostic
	if !errors.As(err, &diag) || diag == nil {
		t.Fatalf("expected *Diagnostic, got %T %v", err, err)
	}
	if !diag.Span.IsSet() {
		t.Fatal("expected span on arity diagnostic")
	}
}
