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

func BenchmarkHashNode_functionBody_warmCache(b *testing.B) {
	fn := benchmarkFunctionBody(b, 200)
	h := New()
	var node ast.Node = fn
	if _, err := h.HashNode(node); err != nil {
		b.Fatalf("prime cache: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := h.HashNode(node); err != nil {
			b.Fatalf("HashNode: %v", err)
		}
	}
}
