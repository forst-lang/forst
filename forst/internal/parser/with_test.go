package parser

import (
	"forst/internal/ast"
	"strings"
	"testing"
)

func TestParseWithStatement(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		validate func(t *testing.T, with ast.WithNode)
	}{
		{
			name: "with shape literal wiring",
			src: `package main
func TestX(t *testing.T) {
	with {
		Logger: &NopLogger {},
		Clock:  &FakeClock { fixedMs: 2000 },
	} {
		result := expireToken(token)
	}
}`,
			validate: func(t *testing.T, with ast.WithNode) {
				shape, ok := with.Wiring.(ast.ShapeNode)
				if !ok {
					t.Fatalf("expected ShapeNode wiring, got %T", with.Wiring)
				}
				if _, ok := shape.Fields["Logger"]; !ok {
					t.Fatal("expected Logger field in wiring")
				}
				if len(with.Body) == 0 {
					t.Fatal("expected non-empty with body")
				}
			},
		},
		{
			name: "with call expression wiring",
			src: `package main
func TestX(t *testing.T) {
	with ciUserApiServices() {
		result := expireToken(token)
	}
}`,
			validate: func(t *testing.T, with ast.WithNode) {
				call, ok := with.Wiring.(ast.FunctionCallNode)
				if !ok {
					t.Fatalf("expected FunctionCallNode wiring, got %T", with.Wiring)
				}
				if call.Function.ID != "ciUserApiServices" {
					t.Fatalf("call: %q", call.Function.ID)
				}
			},
		},
		{
			name: "nested with",
			src: `package main
func TestX(t *testing.T) {
	with ciUserApiServices() {
		with { Clock: &FakeClock { fixedMs: 2000 } } {
			result := expireToken(token)
		}
	}
}`,
			validate: func(t *testing.T, with ast.WithNode) {
				inner := assertNodeType[ast.WithNode](t, with.Body[0], "WithNode")
				if _, ok := inner.Wiring.(ast.ShapeNode); !ok {
					t.Fatalf("inner wiring: %T", inner.Wiring)
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
			fn := assertNodeType[ast.FunctionNode](t, nodes[1], "FunctionNode")
			with := assertNodeType[ast.WithNode](t, fn.Body[0], "WithNode")
			tt.validate(t, with)
		})
	}
}

func TestParseWith_rejectsBareIdentifierWiring(t *testing.T) {
	src := `package main
func f() {
	with ctx {
		x := 1
	}
}`
	err := parseShouldFail(src)
	if err == nil {
		t.Fatal("expected parse error for `with ctx {`")
	}
	if !strings.Contains(err.Error(), "bare identifier") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseWith_rejectsMissingBody(t *testing.T) {
	src := `package main
func f() {
	with { Logger: x }
}`
	err := parseShouldFail(src)
	if err == nil {
		t.Fatal("expected parse error when with body block is missing")
	}
}
