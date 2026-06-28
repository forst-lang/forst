package transformergo

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
	"forst/internal/typechecker"
)

func TestTransformIfIsCondition_minWithParamInTypeGuard(t *testing.T) {
	t.Parallel()
	src := `package main

type Password = String

is (password Password) Strong(min Int) {
	if password is Min(min) {
		println("ok")
	} else {
		println("no")
	}
}
`
	log := ast.SetupTestLogger(nil)
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := typechecker.New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	tr := New(tc, log)

	var guard ast.TypeGuardNode
	found := false
	for _, n := range nodes {
		if g, ok := n.(ast.TypeGuardNode); ok {
			guard = g
			found = true
			break
		}
		if gp, ok := n.(*ast.TypeGuardNode); ok && gp != nil {
			guard = *gp
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected type guard")
	}
	ifNodePtr, ok := guard.Body[0].(*ast.IfNode)
	if !ok {
		ifNodeVal, ok2 := guard.Body[0].(ast.IfNode)
		if !ok2 {
			t.Fatalf("expected IfNode, got %T", guard.Body[0])
		}
		ifNodePtr = &ifNodeVal
	}
	bin, ok := ifNodePtr.Condition.(ast.BinaryExpressionNode)
	if !ok {
		t.Fatalf("expected binary is condition, got %T", ifNodePtr.Condition)
	}
	asn, ok := bin.Right.(ast.AssertionNode)
	if !ok {
		t.Fatalf("expected AssertionNode, got %T", bin.Right)
	}

	if err := tr.restoreScope(guard); err != nil {
		t.Fatalf("restoreScope: %v", err)
	}
	expr, err := tr.transformIfIsCondition(bin.Left, &asn)
	if err != nil {
		t.Fatalf("transformIfIsCondition: %v", err)
	}
	if expr == nil {
		t.Fatal("expected non-nil expr")
	}
}
