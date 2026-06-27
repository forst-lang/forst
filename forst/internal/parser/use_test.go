package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParseUseStatement(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		validate func(t *testing.T, use ast.UseNode)
	}{
		{
			name: "named use",
			src: `package main
func f() {
	use logger: Logger
}`,
			validate: func(t *testing.T, use ast.UseNode) {
				if use.Ident == nil || use.Ident.ID != "logger" {
					t.Fatalf("binding: %+v", use.Ident)
				}
				if use.ContractType.Ident != "Logger" {
					t.Fatalf("contract: %q", use.ContractType.Ident)
				}
			},
		},
		{
			name: "anonymous use",
			src: `package main
func f() {
	use Logger
}`,
			validate: func(t *testing.T, use ast.UseNode) {
				if use.Ident != nil {
					t.Fatalf("expected anonymous use, got ident %q", use.Ident.ID)
				}
				if use.ContractType.Ident != "Logger" {
					t.Fatalf("contract: %q", use.ContractType.Ident)
				}
			},
		},
		{
			name: "use qualified Go type",
			src: `package main
func f() {
	use w: io.Writer
}`,
			validate: func(t *testing.T, use ast.UseNode) {
				if use.Ident == nil || use.Ident.ID != "w" {
					t.Fatalf("binding: %+v", use.Ident)
				}
				if use.ContractType.Ident != "io.Writer" {
					t.Fatalf("contract: %q", use.ContractType.Ident)
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
			use := assertNodeType[ast.UseNode](t, fn.Body[0], "UseNode")
			tt.validate(t, use)
		})
	}
}

func TestParseUseStatement_forbiddenOptionalBinding(t *testing.T) {
	err := parseShouldFail(`package main
func f() {
	use logger?: Logger
}`)
	if err == nil {
		t.Fatal("expected parse error for optional use binding")
	}
}
