package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParseReceiverMethod(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		validate func(t *testing.T, fn ast.FunctionNode)
	}{
		{
			name: "named receiver with body",
			src: `package main
func (l StdLogger) info(msg String) {
}`,
			validate: func(t *testing.T, fn ast.FunctionNode) {
				if fn.Receiver == nil {
					t.Fatal("expected receiver")
				}
				if fn.Receiver.Ident.ID != "l" {
					t.Fatalf("receiver name: got %q", fn.Receiver.Ident.ID)
				}
				if fn.Receiver.Type.Ident != "StdLogger" {
					t.Fatalf("receiver type: got %q", fn.Receiver.Type.Ident)
				}
				if fn.Ident.ID != "info" {
					t.Fatalf("method name: got %q", fn.Ident.ID)
				}
			},
		},
		{
			name: "type-only receiver",
			src: `package main
func (NopLogger) error(msg String) {}`,
			validate: func(t *testing.T, fn ast.FunctionNode) {
				if fn.Receiver == nil {
					t.Fatal("expected receiver")
				}
				if fn.Receiver.Ident.ID != "" {
					t.Fatalf("expected no receiver name, got %q", fn.Receiver.Ident.ID)
				}
				if fn.Receiver.Type.Ident != "NopLogger" {
					t.Fatalf("receiver type: got %q", fn.Receiver.Type.Ident)
				}
			},
		},
		{
			name: "receiver method with return type",
			src: `package main
func (c FakeClock) now(): Int { return 0 }`,
			validate: func(t *testing.T, fn ast.FunctionNode) {
				if len(fn.ReturnTypes) != 1 || fn.ReturnTypes[0].Ident != ast.TypeInt {
					t.Fatalf("return types: %+v", fn.ReturnTypes)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewTestParser(tt.src, nil)
			nodes, err := p.ParseFile()
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			var fn ast.FunctionNode
			for _, n := range nodes {
				if f, ok := n.(ast.FunctionNode); ok {
					fn = f
					break
				}
			}
			tt.validate(t, fn)
		})
	}
}

func TestParseMethodOnlyContractShape(t *testing.T) {
	src := `package main
type Logger = {
	info(msg String)
	error(msg String)
}`
	p := NewTestParser(src, nil)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	td := assertNodeType[ast.TypeDefNode](t, nodes[1], "TypeDefNode")
	shapeExpr, ok := td.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		t.Fatalf("expected TypeDefShapeExpr, got %T", td.Expr)
	}
	infoField := shapeExpr.Shape.Fields["info"]
	if !infoField.IsMethod {
		t.Fatal("expected info to be a method field")
	}
	if len(infoField.MethodParams) != 1 {
		t.Fatalf("expected 1 param, got %d", len(infoField.MethodParams))
	}
}
