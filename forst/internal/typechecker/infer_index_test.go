package typechecker

import (
	"testing"

	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestCheckTypes_sliceIndexing_variableSubscriptAndAssignment(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	src := `package main
func f(xs []String, i Int): String {
	return xs[i]
}
func g(): []String {
	xs := ["a", "b"]
	xs[1] = "x"
	return xs
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
}
