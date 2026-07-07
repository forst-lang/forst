package transformergo

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
	"forst/internal/typechecker"
)

const ifAfterAssignSrc = `package demo

func f(cells []String, a Int): String {
	x0 := cells[a]
	if x0 == "" {
		return ""
	}
	return x0
}
`

func parseIfAfterAssign(t *testing.T) ([]ast.Node, *typechecker.TypeChecker, *ast.IfNode) {
	t.Helper()
	log := ast.SetupTestLogger(nil)
	p := parser.NewTestParser(ifAfterAssignSrc, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := typechecker.New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	fn := findFunctionNodeForTransform(t, nodes, "f")
	ifn, ok := fn.Body[1].(*ast.IfNode)
	if !ok {
		t.Fatalf("body[1] want *IfNode, got %T", fn.Body[1])
	}
	return nodes, tc, ifn
}

func findFunctionNodeForTransform(t *testing.T, nodes []ast.Node, name string) ast.FunctionNode {
	t.Helper()
	for _, n := range nodes {
		fn, ok := n.(ast.FunctionNode)
		if ok && fn.Ident.ID == ast.Identifier(name) {
			return fn
		}
	}
	t.Fatalf("function %q not found", name)
	return ast.FunctionNode{}
}

func TestTransformIfNode_checkerBoundToDifferentAST(t *testing.T) {
	t.Parallel()
	nodesA, tc, _ := parseIfAfterAssign(t)

	log := ast.SetupTestLogger(nil)
	p := parser.NewTestParser(ifAfterAssignSrc, log)
	nodesB, err := p.ParseFile()
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	fnB := findFunctionNodeForTransform(t, nodesB, "f")
	ifNodeB, ok := fnB.Body[1].(*ast.IfNode)
	if !ok {
		t.Fatalf("body[1] want *IfNode, got %T", fnB.Body[1])
	}

	tr := New(tc, log)
	tr.entryNodes = nodesB
	tc.RebindScopes(nodesA, nodesB)
	if _, err = tr.transformIfNode(ifNodeB); err != nil {
		t.Fatalf("transformIfNode with resolve: %v", err)
	}
	_ = nodesA
}

func TestTransformForstFileToGo_reparsedTestIfSucceeds(t *testing.T) {
	t.Parallel()
	nodesA, tc, _ := parseIfAfterAssign(t)

	log := ast.SetupTestLogger(nil)
	p := parser.NewTestParser(ifAfterAssignSrc, log)
	nodesB, err := p.ParseFile()
	if err != nil {
		t.Fatalf("re-parse B: %v", err)
	}

	tr := New(tc, log)
	tr.entryNodes = nodesB
	tc.RebindScopes(nodesA, nodesB)
	if _, err := tr.TransformForstFileToGo(nodesB); err != nil {
		t.Fatalf("TransformForstFileToGo reparsed: %v", err)
	}
	_ = nodesA
}
