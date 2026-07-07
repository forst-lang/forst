package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

const ifAfterAssignSrc = `package demo

func f(cells []String, a Int): String {
	x0 := cells[a]
	if x0 == "" {
		return ""
	}
	return x0
}
`

func findIfAfterAssign(t *testing.T, nodes []ast.Node) *ast.IfNode {
	t.Helper()
	fn := findFunctionNode(t, nodes, "f")
	ifn, ok := fn.Body[1].(*ast.IfNode)
	if !ok {
		t.Fatalf("body[1] want *IfNode, got %T", fn.Body[1])
	}
	return ifn
}

func TestRestoreScope_ifNodeAfterReparse_withoutRebind(t *testing.T) {
	t.Parallel()
	nodesA := parseNodesForTest(t, []byte(ifAfterAssignSrc))
	tc := testTypeChecker(t)
	if err := tc.CheckTypes(nodesA); err != nil {
		t.Fatalf("CheckTypes(A): %v", err)
	}

	nodesB := parseNodesForTest(t, []byte(ifAfterAssignSrc))
	ifNodeB := findIfAfterAssign(t, nodesB)

	err := tc.RestoreScope(ifNodeB)
	if err == nil {
		t.Fatal("expected RestoreScope to fail on re-parsed if without rebind")
	}
	if !strings.Contains(err.Error(), "scope not found") {
		t.Fatalf("RestoreScope error = %q, want scope not found", err)
	}
}

func TestRestoreScope_ifNodeAfterRebind(t *testing.T) {
	t.Parallel()
	nodesA := parseNodesForTest(t, []byte(ifAfterAssignSrc))
	tc := testTypeChecker(t)
	if err := tc.CheckTypes(nodesA); err != nil {
		t.Fatalf("CheckTypes(A): %v", err)
	}

	nodesB := parseNodesForTest(t, []byte(ifAfterAssignSrc))
	tc.RebindScopes(nodesA, nodesB)
	ifNodeB := findIfAfterAssign(t, nodesB)

	if err := tc.RestoreScope(ifNodeB); err != nil {
		t.Fatalf("RestoreScope(ifFromB): %v", err)
	}
	if !tc.HasScopeForNode(ifNodeB) {
		t.Fatal("expected scope registered on re-parsed if after rebind")
	}
}
