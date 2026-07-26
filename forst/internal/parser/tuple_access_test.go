package parser

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/lexer"

	"github.com/sirupsen/logrus"
)

func TestParseExpression_tupleIndexedAccess(t *testing.T) {
	t.Parallel()
	src := "x.0"
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	p := New(toks, "t.ft", log)
	expr := p.parseExpression()
	fa, ok := expr.(ast.FieldAccessNode)
	if !ok {
		t.Fatalf("want FieldAccessNode, got %T", expr)
	}
	if fa.Field.ID != "0" {
		t.Fatalf("want field 0, got %q", fa.Field.ID)
	}
}
