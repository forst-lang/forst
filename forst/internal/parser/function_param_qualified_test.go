package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParseSimpleParameter_qualifiedSiblingTypeNotAssertion(t *testing.T) {
	t.Parallel()
	p := NewTestParser(`ctx runctx.RunContext`, ast.SetupTestLogger(nil))
	param := p.parseSimpleParameter()
	sp, ok := param.(ast.SimpleParamNode)
	if !ok {
		t.Fatalf("got %T", param)
	}
	if sp.Type.Assertion != nil {
		t.Fatalf("qualified sibling type parsed as assertion: %+v", sp.Type)
	}
	if string(sp.Type.Ident) != "runctx.RunContext" {
		t.Fatalf("got ident %q", sp.Type.Ident)
	}
}
