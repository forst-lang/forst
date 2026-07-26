package parser

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"

	"github.com/sirupsen/logrus"
)

func TestParseFunction_variadicParameter(t *testing.T) {
	t.Parallel()
	src := "package main\nfunc sum(nums ...Int): Int { return 0 }\n"
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	fn := findFunctionByName(t, nodes, "sum")
	sp, ok := fn.Params[0].(ast.SimpleParamNode)
	if !ok || !sp.Variadic || sp.Type.Ident != ast.TypeInt {
		t.Fatalf("want variadic ...Int param, got %#v", fn.Params[0])
	}
}

func TestParseFunction_variadicMustBeLast(t *testing.T) {
	t.Parallel()
	src := "package main\nfunc bad(a ...Int, b Int) {}\n"
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	defer func() {
		if recover() == nil {
			t.Fatal("expected parse error for non-trailing variadic param")
		}
	}()
	_, _ = New(toks, "t.ft", log).ParseFile()
}

func findFunctionByName(t *testing.T, nodes []ast.Node, name string) ast.FunctionNode {
	t.Helper()
	for _, n := range nodes {
		if fn, ok := n.(ast.FunctionNode); ok && fn.Ident.ID == ast.Identifier(name) {
			return fn
		}
	}
	t.Fatalf("function not found: %s", name)
	return ast.FunctionNode{}
}
