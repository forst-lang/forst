package hasher

import (
	"fmt"
	"testing"

	"forst/internal/ast"
)

func benchmarkFunctionBody(b *testing.B, stmtCount int) ast.FunctionNode {
	body := make([]ast.Node, stmtCount)
	for i := 0; i < stmtCount; i++ {
		body[i] = ast.AssignmentNode{
			LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(fmt.Sprintf("v%d", i))}}},
			RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: int64(i)}},
		}
	}
	return ast.FunctionNode{
		Ident: ast.Ident{ID: "benchFn"},
		Body:  body,
	}
}

func BenchmarkHashNode_functionBody(b *testing.B) {
	fn := benchmarkFunctionBody(b, 200)
	h := New()
	if _, err := h.HashNode(fn); err != nil {
		b.Fatalf("warm HashNode: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := h.HashNode(fn); err != nil {
			b.Fatalf("HashNode: %v", err)
		}
	}
}

func BenchmarkHashNode_functionBody_walkMemo(b *testing.B) {
	// Same ast.Node interface value referenced many times in one tree; walk-local memo
	// should hit on repeated identities within a single HashNode call.
	sharedLit := ast.IntLiteralNode{Value: 42}
	var shared ast.ExpressionNode = sharedLit
	body := make([]ast.Node, 200)
	for i := range body {
		body[i] = ast.ReturnNode{
			Values: []ast.ExpressionNode{shared, shared},
		}
	}
	fn := ast.FunctionNode{
		Ident: ast.Ident{ID: "benchFn"},
		Body:  body,
	}
	h := New()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := h.HashNode(fn); err != nil {
			b.Fatalf("HashNode: %v", err)
		}
	}
}
