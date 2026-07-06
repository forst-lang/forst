package typechecker

import (
	"io"
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func testTypeChecker(t *testing.T) *TypeChecker {
	t.Helper()
	log := logrus.New()
	log.SetOutput(io.Discard)
	return New(log, false)
}

func parseNodesForTest(t *testing.T, src []byte) []ast.Node {
	t.Helper()
	log := logrus.New()
	log.SetOutput(io.Discard)
	toks := lexer.New(src, "test.ft", log).Lex()
	nodes, err := parser.New(toks, "test.ft", log).ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return nodes
}

func TestRestoreScope_distinctEnsureSubjectsDistinctScopes(t *testing.T) {
	t.Parallel()
	src := []byte(`package demo

type Named = { name: String }

func F(a: Named, b: Named): Void {
	ensure a.name is Min(1) {}
	ensure b.name is Min(1) {}
}
`)
	nodes := parseNodesForTest(t, src)
	tc := testTypeChecker(t)
	if err := tc.CollectTypes(nodes); err != nil {
		t.Fatalf("CollectTypes: %v", err)
	}
	var ensureScopes int
	for _, scope := range tc.scopeStack.scopes {
		if scope.Node == nil {
			continue
		}
		if _, ok := (*scope.Node).(ast.EnsureNode); ok {
			ensureScopes++
		}
	}
	if ensureScopes < 2 {
		t.Fatalf("expected at least 2 ensure scopes, got %d", ensureScopes)
	}
}
