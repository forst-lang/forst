package parser

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestScopeStack_pushPopAndGlobals(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	ss := NewScopeStack(log)
	if ss.GlobalScope() == nil || ss.CurrentScope() == nil {
		t.Fatal("expected non-nil global/current scope")
	}
	if !ss.GlobalScope().IsGlobal {
		t.Fatal("expected global scope flag")
	}
	scope := NewScope("f", false, false, log)
	scope.DefineVariable("x", ast.TypeNode{Ident: ast.TypeInt})
	ss.PushScope(scope)
	if ss.CurrentScope().FunctionName != "f" {
		t.Fatalf("current scope function mismatch: %q", ss.CurrentScope().FunctionName)
	}
	if _, ok := ss.CurrentScope().Variables["x"]; !ok {
		t.Fatal("expected variable in pushed scope")
	}
	ss.PopScope()
	if ss.CurrentScope() != ss.GlobalScope() {
		t.Fatal("expected current scope to be global after pop")
	}
}

