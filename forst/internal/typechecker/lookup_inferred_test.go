package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestLookupInferredType_requireInferred_registered(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	n := ast.IntLiteralNode{Value: 7}
	hash, err := tc.Hasher.HashNode(n)
	if err != nil {
		t.Fatal(err)
	}
	tc.Types[hash] = []ast.TypeNode{{Ident: ast.TypeInt}}
	got, err := tc.LookupInferredType(n, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Ident != ast.TypeInt {
		t.Fatalf("got %+v", got)
	}
}

func TestLookupInferredType_emptySlice_requireInferred_errors(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	n := ast.IntLiteralNode{Value: 1}
	hash, err := tc.Hasher.HashNode(n)
	if err != nil {
		t.Fatal(err)
	}
	tc.Types[hash] = []ast.TypeNode{}
	_, err = tc.LookupInferredType(n, true)
	if err == nil || !strings.Contains(err.Error(), "implicit type") {
		t.Fatalf("got %v", err)
	}
}

func TestLookupInferredType_missing_requireInferred_errors(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	_, err := tc.LookupInferredType(ast.IntLiteralNode{Value: 2}, true)
	if err == nil || !strings.Contains(err.Error(), "no registered type") {
		t.Fatalf("got %v", err)
	}
}

func TestLookupInferredType_missing_notRequired_nil(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	got, err := tc.LookupInferredType(ast.IntLiteralNode{Value: 3}, false)
	if err != nil || got != nil {
		t.Fatalf("got %v %v", got, err)
	}
}
