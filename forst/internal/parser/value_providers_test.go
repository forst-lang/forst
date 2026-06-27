package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParseValue_pointerCompositeLiteral(t *testing.T) {
	src := `package main
func f() {
	with {
		Logger: &NopLogger {},
		Clock:  &FakeClock { fixedMs: 2000 },
	} {
		x := 1
	}
}`
	p := NewTestParser(src, nil)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	fn := assertNodeType[ast.FunctionNode](t, nodes[1], "FunctionNode")
	with := assertNodeType[ast.WithNode](t, fn.Body[0], "WithNode")
	shape, ok := with.Wiring.(ast.ShapeNode)
	if !ok {
		t.Fatalf("wiring type %T", with.Wiring)
	}

	loggerRef := assertNodeType[ast.ReferenceNode](t, shape.Fields["Logger"].Node, "ReferenceNode")
	loggerShape := assertNodeType[ast.ShapeNode](t, loggerRef.Value, "ShapeNode")
	if loggerShape.BaseType == nil || string(*loggerShape.BaseType) != "NopLogger" {
		t.Fatalf("Logger base type: %v", loggerShape.BaseType)
	}

	clockRef := assertNodeType[ast.ReferenceNode](t, shape.Fields["Clock"].Node, "ReferenceNode")
	clockShape := assertNodeType[ast.ShapeNode](t, clockRef.Value, "ShapeNode")
	if clockShape.BaseType == nil || string(*clockShape.BaseType) != "FakeClock" {
		t.Fatalf("Clock base type: %v", clockShape.BaseType)
	}
	fixedMs, ok := clockShape.Fields["fixedMs"].Node.(ast.IntLiteralNode)
	if !ok || fixedMs.Value != 2000 {
		t.Fatalf("fixedMs field: %v", clockShape.Fields["fixedMs"].Node)
	}
}
