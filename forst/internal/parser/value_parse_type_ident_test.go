package parser

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestParseTypeIdent_identifierToken(t *testing.T) {
	t.Parallel()
	p := New(
		[]ast.Token{
			{Type: ast.TokenIdentifier, Value: "User"},
			{Type: ast.TokenEOF, Value: ""},
		},
		"type_ident.ft",
		logrus.New(),
	)
	got := p.parseTypeIdent()
	if got == nil {
		t.Fatal("expected type identifier")
	}
	if string(*got) != "User" {
		t.Fatalf("expected User, got %q", string(*got))
	}
}

