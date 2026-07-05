package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParseType_interfaceEmpty(t *testing.T) {
	t.Parallel()
	logger := ast.SetupTestLogger(nil)
	p := setupParser([]ast.Token{
		{Type: ast.TokenInterface, Value: "interface"},
		{Type: ast.TokenLBrace, Value: "{"},
		{Type: ast.TokenRBrace, Value: "}"},
		{Type: ast.TokenEOF},
	}, logger)
	typ := p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
	if typ.Ident != ast.TypeObject {
		t.Fatalf("got %v", typ.Ident)
	}
}
