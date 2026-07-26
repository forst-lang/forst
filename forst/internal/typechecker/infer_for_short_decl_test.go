package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestInferForNode_shortDeclVisibleToFollowingIf(t *testing.T) {
	t.Parallel()
	src := `package main

func winnerOrEmpty(cells []String): String {
	for i := 0; i < 8; i++ {
		w := "x"
		if w != "" {
			return w
		}
	}
	return ""
}
`
	p := parser.NewTestParser(src, ast.SetupTestLogger(nil))
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(logrus.New(), false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
}
