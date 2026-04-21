package transformergo

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestMapIndexExprCacheKey_equalForDuplicateReads(t *testing.T) {
	src := `package main

func main() {
	m := map[String]Int{ "a": 1 }
	a := m["a"]
	b := m["a"]
	ensure a is Ok()
	ensure b is Ok()
	println(string(a))
	println(string(b))
}
`
	log := logrus.New()
	log.SetOutput(nil)
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	var idxs []ast.IndexExpressionNode
	var walk func(ast.Node)
	walk = func(n ast.Node) {
		if n == nil {
			return
		}
		switch x := n.(type) {
		case ast.IndexExpressionNode:
			idxs = append(idxs, x)
		case ast.FunctionNode:
			for _, st := range x.Body {
				walk(st)
			}
		case ast.AssignmentNode:
			for _, rv := range x.RValues {
				walk(rv)
			}
		}
	}
	for _, n := range nodes {
		if fn, ok := n.(ast.FunctionNode); ok && string(fn.Ident.ID) == "main" {
			for _, st := range fn.Body {
				walk(st)
			}
		}
	}
	if len(idxs) < 2 {
		t.Fatalf("want 2 index exprs, got %d", len(idxs))
	}
	k0 := mapIndexExprCacheKey(idxs[0])
	k1 := mapIndexExprCacheKey(idxs[1])
	if k0 != k1 {
		t.Fatalf("cache keys differ: %q vs %q (idx0.String=%q idx1.String=%q)",
			k0, k1, idxs[0].String(), idxs[1].String())
	}
}
