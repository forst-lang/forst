package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestRestoreScope_typeGuardBody_matchSymbolScope(t *testing.T) {
	t.Parallel()
	src := []byte(`package demo

type Password = String

is (password Password) Strong(min Int) {
	ensure password is Min(min)
	if password is Min(min) {
		println("ok")
	} else if password is Max(100) {
		println("mid")
	} else {
		println("no")
	}
}
`)
	nodes := parseNodesForTest(t, src)
	tc := testTypeChecker(t)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}

	var guardNode ast.Node
	var guard ast.TypeGuardNode
	for _, n := range nodes {
		switch g := n.(type) {
		case ast.TypeGuardNode:
			guardNode = n
			guard = g
		case *ast.TypeGuardNode:
			if g != nil {
				guardNode = n
				guard = *g
			}
		}
		if guardNode != nil {
			break
		}
	}
	if guardNode == nil {
		t.Fatal("type guard node not found")
	}

	regGuard, ok := tc.ScopeNodeForTypeGuard(guard.Ident)
	if !ok || regGuard != guardNode {
		t.Fatalf("ScopeNodeForTypeGuard: got %v ok=%v want parse node %v", regGuard, ok, guardNode)
	}

	t.Run("guard param scope", func(t *testing.T) {
		if err := tc.RestoreScope(guardNode); err != nil {
			t.Fatalf("RestoreScope guard: %v", err)
		}
		sym, ok := tc.CurrentScope().LookupVariable(ast.Identifier("password"))
		if !ok || sym.Scope == nil {
			t.Fatal("password not in guard scope")
		}
		if err := tc.RestoreScope(guardNode); err != nil {
			t.Fatalf("RestoreScope guard again: %v", err)
		}
		if tc.CurrentScope() != sym.Scope {
			t.Fatal("RestoreScope guard scope pointer != symbol scope")
		}
	})

	var ifNode *ast.IfNode
	var ensureNode ast.Node
	for _, bodyNode := range guard.Body {
		switch n := bodyNode.(type) {
		case ast.EnsureNode:
			ensureNode = bodyNode
		case *ast.IfNode:
			ifNode = n
		}
	}
	if ensureNode == nil {
		t.Fatal("ensure node not found in guard body")
	}
	if ifNode == nil {
		t.Fatal("if node not found in guard body")
	}

	t.Run("ensure scope", func(t *testing.T) {
		if err := tc.RestoreScope(ensureNode); err != nil {
			t.Fatalf("RestoreScope ensure: %v", err)
		}
	})

	t.Run("if then", func(t *testing.T) {
		if err := tc.RestoreScope(ifNode); err != nil {
			t.Fatalf("RestoreScope if: %v", err)
		}
	})

	if len(ifNode.ElseIfs) > 0 {
		t.Run("else if", func(t *testing.T) {
			if err := tc.RestoreScope(&ifNode.ElseIfs[0]); err != nil {
				t.Fatalf("RestoreScope else-if: %v", err)
			}
		})
	}

	if ifNode.Else != nil {
		t.Run("else", func(t *testing.T) {
			if err := tc.RestoreScope(ifNode.Else); err != nil {
				t.Fatalf("RestoreScope else: %v", err)
			}
		})
	}

	// Value copies must not match registered scopes.
	t.Run("unboxed guard value misses", func(t *testing.T) {
		if err := tc.RestoreScope(ast.Node(guard)); err == nil {
			t.Fatal("expected RestoreScope on unboxed guard value to fail")
		}
	})
	if en, ok := ensureNode.(ast.EnsureNode); ok {
		t.Run("unboxed ensure value misses", func(t *testing.T) {
			if err := tc.RestoreScope(en); err == nil {
				t.Fatal("expected RestoreScope on unboxed ensure value to fail")
			}
		})
	}
}
