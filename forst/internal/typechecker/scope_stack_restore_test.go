package typechecker

import (
	"fmt"
	"io"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestRestoreScope_nilGlobal_afterCheckTypes(t *testing.T) {
	t.Parallel()
	src := []byte(`package demo

func main() {}
`)
	nodes := parseNodesForTest(t, src)
	tc := testTypeChecker(t)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	if err := tc.RestoreScope(nil); err != nil {
		t.Fatalf("RestoreScope(nil): %v", err)
	}
	if !tc.CurrentScope().IsGlobal() {
		t.Fatal("expected global scope after RestoreScope(nil)")
	}
}

func TestRestoreScope_reusesCachedHash(t *testing.T) {
	t.Parallel()
	var body []ast.Node
	for i := 0; i < 50; i++ {
		body = append(body, ast.AssignmentNode{
			LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(fmt.Sprintf("v%d", i))}}},
			RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: int64(i)}},
		})
	}
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "f"}, Body: body}
	nodes := []ast.Node{fn}
	log := logrus.New()
	log.SetOutput(io.Discard)
	tc := New(log, false)
	if err := tc.CollectTypes(nodes); err != nil {
		t.Fatalf("CollectTypes: %v", err)
	}
	scopeNode := nodes[0]
	want, ok := tc.scopeStack.findScope(scopeNode)
	if !ok {
		t.Fatal("scope not found for fn after collect")
	}
	for i := 0; i < 1000; i++ {
		if err := tc.RestoreScope(scopeNode); err != nil {
			t.Fatalf("RestoreScope iteration %d: %v", i, err)
		}
		if tc.scopeStack.current != want {
			t.Fatalf("iteration %d: scope pointer changed", i)
		}
	}
}
