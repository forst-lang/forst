package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParseType_extraForms(t *testing.T) {
	t.Parallel()
	logger := ast.SetupTestLogger(nil)
	tests := []struct {
		name string
		src  string
		check func(t *testing.T, typ ast.TypeNode)
	}{
		{
			name: "array_paren_form",
			src:  "Array(Int)",
			check: func(t *testing.T, typ ast.TypeNode) {
				if typ.Ident != ast.TypeArray {
					t.Fatalf("got %v", typ.Ident)
				}
			},
		},
		{
			name: "map_bracket_form",
			src:  "map[Int]String",
			check: func(t *testing.T, typ ast.TypeNode) {
				if typ.Ident != ast.TypeMap || len(typ.TypeParams) != 2 {
					t.Fatalf("got %+v", typ)
				}
			},
		},
		{
			name: "chan_type",
			src:  "chan Int",
			check: func(t *testing.T, typ ast.TypeNode) {
				if typ.Ident != ast.TypeChannel {
					t.Fatalf("got %v", typ.Ident)
				}
			},
		},
		{
			name: "result_type",
			src:  "Result(Int, Error)",
			check: func(t *testing.T, typ ast.TypeNode) {
				if typ.Ident != ast.TypeResult {
					t.Fatalf("got %v", typ.Ident)
				}
			},
		},
		{
			name: "tuple_type",
			src:  "Tuple(Int, String)",
			check: func(t *testing.T, typ ast.TypeNode) {
				if typ.Ident != ast.TypeTuple || len(typ.TypeParams) != 2 {
					t.Fatalf("got %+v", typ)
				}
			},
		},
		{
			name: "shape_assertion_type",
			src:  "{ x: Int }",
			check: func(t *testing.T, typ ast.TypeNode) {
				if typ.Assertion == nil || typ.Assertion.Constraints[0].Name != "Match" {
					t.Fatalf("got %+v", typ)
				}
			},
		},
		{
			name: "void_builtin",
			src:  "Void",
			check: func(t *testing.T, typ ast.TypeNode) {
				if typ.Ident != ast.TypeVoid {
					t.Fatalf("got %v", typ.Ident)
				}
			},
		},
		{
			name: "error_builtin",
			src:  "Error",
			check: func(t *testing.T, typ ast.TypeNode) {
				if typ.Ident != ast.TypeError {
					t.Fatalf("got %v", typ.Ident)
				}
			},
		},
		{
			name: "qualified_type",
			src:  "pkg.User",
			check: func(t *testing.T, typ ast.TypeNode) {
				if string(typ.Ident) != "pkg.User" {
					t.Fatalf("got %v", typ.Ident)
				}
			},
		},
		{
			name: "double_pointer",
			src:  "**Int",
			check: func(t *testing.T, typ ast.TypeNode) {
				if typ.Ident != ast.TypePointer || typ.TypeParams[0].Ident != ast.TypePointer {
					t.Fatalf("got %+v", typ)
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := NewTestParser(tc.src, logger)
			typ := p.parseType(TypeIdentOpts{AllowLowercaseTypes: true})
			tc.check(t, typ)
		})
	}
}

func TestParseParameter_destructuredAndInvalid(t *testing.T) {
	t.Parallel()
	logger := ast.SetupTestLogger(nil)

	p := NewTestParser("{a, b}: { a: Int, b: Int }", logger)
	param := p.parseParameter()
	if _, ok := param.(ast.DestructuredParamNode); !ok {
		t.Fatalf("got %T", param)
	}

	p2 := setupParser([]ast.Token{{Type: ast.TokenPlus, Value: "+"}, {Type: ast.TokenEOF}}, logger)
	defer func() {
		if recover() == nil {
			t.Fatal("expected parse error for invalid parameter")
		}
	}()
	_ = p2.parseParameter()
}

func TestBinaryPrecedence_unknownOperator(t *testing.T) {
	t.Parallel()
	if got := binaryPrecedence(ast.TokenArrow); got != 0 {
		t.Fatalf("got %d", got)
	}
}
