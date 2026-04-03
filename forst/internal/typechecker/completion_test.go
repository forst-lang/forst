package typechecker

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestListFieldNamesForType_builtinString(t *testing.T) {
	log := logrus.New()
	tc := New(log, false)
	names := tc.ListFieldNamesForType(ast.TypeNode{Ident: ast.TypeString})
	if len(names) != 1 || names[0] != "len" {
		t.Fatalf("got %#v", names)
	}
}
