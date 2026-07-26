package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParse_makeNew_typeArguments(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		src  string
		want int // arg count
	}{
		{name: "make slice", src: "make(Array(Int), 10)", want: 2},
		{name: "make slice cap", src: "make(Array(Int), 10, 20)", want: 3},
		{name: "make map", src: "make(map[String]Int)", want: 1},
		{name: "make map hint", src: "make(map[String]Int, 8)", want: 2},
		{name: "new pointer", src: "new(Int)", want: 1},
		{name: "new star type", src: "new(*Int)", want: 1},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := NewTestParser("package main\n\nfunc main() { xs := "+tc.src+" }\n", ast.SetupTestLogger(nil))
			nodes, err := p.ParseFile()
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			fn := nodes[len(nodes)-1].(ast.FunctionNode)
			assign := assertNodeType[ast.AssignmentNode](t, fn.Body[0], "ast.AssignmentNode")
			if !assign.IsShort {
				t.Fatal("expected short assignment")
			}
			call := assertNodeType[ast.FunctionCallNode](t, assign.RValues[0], "ast.FunctionCallNode")
			if len(call.Arguments) != tc.want {
				t.Fatalf("args: got %d want %d", len(call.Arguments), tc.want)
			}
			te, ok := call.Arguments[0].(ast.TypeExpressionNode)
			if !ok {
				t.Fatalf("first arg: got %T want TypeExpressionNode", call.Arguments[0])
			}
			if te.Type.Ident == "" {
				t.Fatal("expected non-empty type on first argument")
			}
		})
	}
}

func TestParse_makeNew_rejectsValueFirstArg(t *testing.T) {
	t.Parallel()
	p := NewTestParser("package main\n\nfunc main() { make(1) }\n", ast.SetupTestLogger(nil))
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		_, _ = p.ParseFile()
	}()
	if recovered == nil {
		t.Fatal("expected parse failure when make first arg is not a type")
	}
}
