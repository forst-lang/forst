package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestScopeOwnerRegistry_populatedAtCollect(t *testing.T) {
	t.Parallel()
	src := []byte(`package demo

type Logger = {info(msg String)}

type NopLogger = {}

func (NopLogger) info(msg String) {}

type Name = String

is (n Name) Active(min Int) {
	ensure n is Min(min)
}

func host() {
	with { Logger: &NopLogger{} } {
		use logger: Logger
	}
}
`)
	nodes := parseNodesForTest(t, src)
	tc := testTypeChecker(t)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}

	var fnNode ast.Node
	var guardNode ast.Node
	for _, n := range nodes {
		switch x := n.(type) {
		case ast.FunctionNode:
			if x.Ident.ID == "host" {
				fnNode = n
			}
		case *ast.FunctionNode:
			if x != nil && x.Ident.ID == "host" {
				fnNode = n
			}
		case ast.TypeGuardNode:
			if x.Ident == "Active" {
				guardNode = n
			}
		case *ast.TypeGuardNode:
			if x != nil && x.Ident == "Active" {
				guardNode = n
			}
		}
	}
	if fnNode == nil {
		t.Fatal("host function node not found")
	}
	if guardNode == nil {
		t.Fatal("Active type guard node not found")
	}

	regFn, ok := tc.ScopeNodeForFunction(ast.Identifier("host"))
	if !ok {
		t.Fatal("ScopeNodeForFunction(host): missing")
	}
	if !tc.HasScopeForNode(fnNode) {
		t.Fatal("parse host node missing registered scope")
	}
	regGuard, ok := tc.ScopeNodeForTypeGuard(ast.Identifier("Active"))
	if !ok {
		t.Fatal("ScopeNodeForTypeGuard(Active): missing")
	}
	if !tc.HasScopeForNode(guardNode) {
		t.Fatal("parse Active guard node missing registered scope")
	}

	if err := tc.RestoreScope(regFn); err != nil {
		t.Fatalf("RestoreScope via registry fn: %v", err)
	}
	if err := tc.RestoreScope(regGuard); err != nil {
		t.Fatalf("RestoreScope via registry guard: %v", err)
	}
}

func TestScopeOwnerRegistry_resetOnCollectTypes(t *testing.T) {
	t.Parallel()
	src := []byte(`package demo

func f() {}
`)
	nodes := parseNodesForTest(t, src)
	tc := testTypeChecker(t)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes re-run: %v", err)
	}
	second, ok := tc.ScopeNodeForFunction(ast.Identifier("f"))
	if !ok {
		t.Fatal("expected ScopeNodeForFunction after re-run CheckTypes")
	}
	if err := tc.RestoreScope(second); err != nil {
		t.Fatalf("RestoreScope after re-run CheckTypes: %v", err)
	}
}
