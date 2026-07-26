package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParseConst_single(t *testing.T) {
	t.Parallel()
	src := `package main

const Pi = 3
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	group, ok := nodes[1].(ast.ConstGroupNode)
	if !ok {
		t.Fatalf("want ConstGroupNode, got %T", nodes[1])
	}
	if len(group.Specs) != 1 {
		t.Fatalf("specs = %d", len(group.Specs))
	}
	spec := group.Specs[0]
	if spec.Name.ID != "Pi" {
		t.Fatalf("name = %q", spec.Name.ID)
	}
	if _, ok := spec.Value.(ast.IntLiteralNode); !ok {
		t.Fatalf("value = %T", spec.Value)
	}
}

func TestParseConst_groupWithIota(t *testing.T) {
	t.Parallel()
	src := `package main

const (
  A = iota
  B
  C
)
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	group, ok := nodes[1].(ast.ConstGroupNode)
	if !ok {
		t.Fatalf("want ConstGroupNode, got %T", nodes[1])
	}
	if len(group.Specs) != 3 {
		t.Fatalf("specs = %d", len(group.Specs))
	}
	if _, ok := group.Specs[0].Value.(ast.IotaLiteralNode); !ok {
		t.Fatalf("A value = %T", group.Specs[0].Value)
	}
	if group.Specs[1].Value != nil {
		t.Fatalf("B should repeat previous expr, got %v", group.Specs[1].Value)
	}
	if group.Specs[2].Value != nil {
		t.Fatalf("C should repeat previous expr, got %v", group.Specs[2].Value)
	}
}

func TestParseConst_shiftIotaGroup(t *testing.T) {
	t.Parallel()
	src := `package main

const (
  FlagNone = 1 << iota
  FlagRead
)
`
	nodes, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	group := nodes[1].(ast.ConstGroupNode)
	bin, ok := group.Specs[0].Value.(ast.BinaryExpressionNode)
	if !ok {
		t.Fatalf("value = %T", group.Specs[0].Value)
	}
	if bin.Operator != ast.TokenLShift {
		t.Fatalf("op = %q", bin.Operator)
	}
}
