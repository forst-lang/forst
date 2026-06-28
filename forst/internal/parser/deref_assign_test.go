package parser

import (
	"forst/internal/ast"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestParseBlockStatement_derefAssign_doubleStar(t *testing.T) {
	src := `package main
func main() {
	**pp = 20
}
`
	p := NewTestParser(src, logrus.New())
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	var fn ast.FunctionNode
	for _, n := range nodes {
		if f, ok := n.(ast.FunctionNode); ok {
			fn = f
			break
		}
	}
	if len(fn.Body) != 1 {
		t.Fatalf("body len: got %d want 1", len(fn.Body))
	}
	if _, ok := fn.Body[0].(ast.AssignmentNode); !ok {
		t.Fatalf("want AssignmentNode, got %T", fn.Body[0])
	}
}
