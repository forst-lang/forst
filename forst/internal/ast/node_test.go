package ast

import "testing"

func TestNodeKind_distinct_core_kinds(t *testing.T) {
	if NodeKindFunction == NodeKindIf || NodeKindReturn == NodeKindVariable {
		t.Fatal("NodeKind constants should differ")
	}
	if string(NodeKindFunction) != "Function" {
		t.Fatal(NodeKindFunction)
	}
}
