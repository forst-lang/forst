package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestRestoreScope_withBody_matchSymbolScope(t *testing.T) {
	t.Parallel()
	src := []byte(`package demo

type Logger = {info(msg String)}

type NopLogger = {}

func (NopLogger) info(msg String) {}

func host() {
	with { Logger: &NopLogger{} } {
		use logger: Logger
		logger.info("hi")
	}
}
`)
	nodes := parseNodesForTest(t, src)
	tc := testTypeChecker(t)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}

	var withNode ast.Node
	var with ast.WithNode
	for _, n := range nodes {
		fn, ok := n.(ast.FunctionNode)
		if !ok || fn.Ident.ID != "host" {
			continue
		}
		for _, stmt := range fn.Body {
			if w, ok := stmt.(ast.WithNode); ok {
				withNode = stmt
				with = w
				break
			}
		}
	}
	if withNode == nil {
		t.Fatal("with node not found in host body")
	}

	if err := tc.RestoreScope(withNode); err != nil {
		t.Fatalf("RestoreScope with: %v", err)
	}
	sym, ok := tc.CurrentScope().LookupVariable(ast.Identifier("logger"))
	if !ok || sym.Scope == nil {
		t.Fatal("logger not in with scope after RestoreScope")
	}
	if err := tc.RestoreScope(withNode); err != nil {
		t.Fatalf("RestoreScope with again: %v", err)
	}
	if tc.CurrentScope() != sym.Scope {
		t.Fatal("RestoreScope with scope pointer != symbol scope")
	}

	t.Run("unboxed with value misses", func(t *testing.T) {
		if err := tc.RestoreScope(ast.Node(with)); err == nil {
			t.Fatal("expected RestoreScope on unboxed with value to fail")
		}
	})
}
