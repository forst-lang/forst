package parser

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"
)

func TestParseGenericInstantiationCall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		src   string
		check func(t *testing.T, expr ast.ExpressionNode)
	}{
		{
			name: "explicitTypeArgs",
			src:  `identity[Int](42)`,
			check: func(t *testing.T, expr ast.ExpressionNode) {
				t.Helper()
				call, ok := expr.(ast.FunctionCallNode)
				if !ok {
					t.Fatalf("expected FunctionCallNode, got %T", expr)
				}
				if call.Function.ID != "identity" {
					t.Fatalf("function: %s", call.Function.ID)
				}
				if len(call.TypeArgs) != 1 || call.TypeArgs[0].Ident != ast.TypeInt {
					t.Fatalf("TypeArgs: %+v", call.TypeArgs)
				}
				if len(call.Arguments) != 1 {
					t.Fatalf("Arguments: %+v", call.Arguments)
				}
			},
		},
		{
			name: "indexStillWorks",
			src:  `xs[0]`,
			check: func(t *testing.T, expr ast.ExpressionNode) {
				t.Helper()
				idx, ok := expr.(ast.IndexExpressionNode)
				if !ok {
					t.Fatalf("expected IndexExpressionNode, got %T", expr)
				}
				if _, ok := idx.Target.(ast.VariableNode); !ok {
					t.Fatalf("target: %T", idx.Target)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := New(tokenize(tt.src), "test.ft", nil)
			tt.check(t, p.parseExpression())
		})
	}
}

func tokenize(src string) []ast.Token {
	return lexer.New([]byte(src), "test.ft", ast.SetupTestLogger(nil)).Lex()
}
