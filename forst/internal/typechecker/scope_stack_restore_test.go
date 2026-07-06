package typechecker

import (
	"fmt"
	"io"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

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
	log := logrus.New()
	log.SetOutput(io.Discard)
	tc := New(log, false)
	if err := tc.CollectTypes([]ast.Node{fn}); err != nil {
		t.Fatalf("CollectTypes: %v", err)
	}
	want, ok := tc.scopeStack.findScope(fn)
	if !ok {
		t.Fatal("scope not found for fn after collect")
	}
	for i := 0; i < 1000; i++ {
		if err := tc.RestoreScope(fn); err != nil {
			t.Fatalf("RestoreScope iteration %d: %v", i, err)
		}
		if tc.scopeStack.current != want {
			t.Fatalf("iteration %d: scope pointer changed", i)
		}
	}
}
