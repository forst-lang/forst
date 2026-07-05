package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParseFunction_sliceStringParam(t *testing.T) {
	p := NewTestParser(`func ParseArgs(args []String): ParsedArgs { return ParsedArgs{} }`, ast.SetupTestLogger(nil))
	fn := p.parseFunctionDefinition()
	sp, ok := fn.Params[0].(ast.SimpleParamNode)
	if !ok {
		t.Fatalf("param %T", fn.Params[0])
	}
	t.Logf("param type %+v", sp.Type)
	if sp.Type.Ident != ast.TypeArray {
		t.Fatalf("want array param, got %q", sp.Type.Ident)
	}
}
