package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestVariadicFunction_signatureRegistered(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	src := "package main\nfunc f(xs ...Int): Int { return 0 }\n"
	tc := typecheckSource(t, dir, src)
	sig := tc.Functions[ast.Identifier("f")]
	if len(sig.Parameters) != 1 || !sig.Parameters[0].Variadic {
		t.Fatalf("want variadic signature, got %#v", sig.Parameters)
	}
}

func TestVariadicFunction_callWithInlineArgs(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	src := "package main\nfunc f(xs ...Int): Int { return 0 }\nfunc main() { f(1, 2, 3) }\n"
	typecheckSource(t, dir, src)
}

func TestVariadicFunction_callWithSpread(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	src := "package main\nfunc f(xs ...Int): Int { return 0 }\nfunc main() {\n  var xs: Array(Int)\n  f(xs...)\n}\n"
	typecheckSource(t, dir, src)
}

func TestVariadicFunction_rejectsTooFewFixedArgs(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)
	src := "package main\nfunc f(a Int, rest ...Int): Int { return 0 }\nfunc main() { f() }\n"
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = dir
	if err := tc.CheckTypes(nodes); err == nil {
		t.Fatal("expected arity error")
	}
}
