package parser

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"

	"github.com/sirupsen/logrus"
)

func TestParseGenericInstantiationCall_explicitTypeArgs(t *testing.T) {
	t.Parallel()
	tokens := tokenize(`identity[Int](42)`)
	p := New(tokens, "test.ft", nil)
	expr := p.parseExpression()
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
}

func TestParseGenericInstantiationCall_indexStillWorks(t *testing.T) {
	t.Parallel()
	tokens := tokenize(`xs[0]`)
	p := New(tokens, "test.ft", nil)
	expr := p.parseExpression()
	idx, ok := expr.(ast.IndexExpressionNode)
	if !ok {
		t.Fatalf("expected IndexExpressionNode, got %T", expr)
	}
	if _, ok := idx.Target.(ast.VariableNode); !ok {
		t.Fatalf("target: %T", idx.Target)
	}
}

func tokenize(src string) []ast.Token {
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	return lexer.New([]byte(src), "test.ft", log).Lex()
}
