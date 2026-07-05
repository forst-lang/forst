package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParseFunction_goTestingTParam(t *testing.T) {
	t.Parallel()
	p := NewTestParser(`func TestDemo(t *testing.T) {}`, ast.SetupTestLogger(nil))
	fn := p.parseFunctionDefinition()
	sp, ok := fn.Params[0].(ast.SimpleParamNode)
	if !ok {
		t.Fatalf("param %T", fn.Params[0])
	}
	if sp.Type.Ident != ast.TypePointer {
		t.Fatalf("expected pointer param for Go testing convention, got %+v", sp.Type)
	}
	if len(sp.Type.TypeParams) != 1 || sp.Type.TypeParams[0].Ident != ast.TypeIdent("testing.T") {
		t.Fatalf("expected *testing.T param, got %+v", sp.Type)
	}
}
