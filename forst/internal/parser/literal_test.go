package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestParseLiteral_primitives_and_nil(t *testing.T) {
	logger := ast.SetupTestLogger(nil)

	t.Run("string_literal", func(t *testing.T) {
		toks := []ast.Token{
			{Type: ast.TokenStringLiteral, Value: `"ab"`},
			{Type: ast.TokenEOF},
		}
		p := setupParser(toks, logger)
		lit := p.parseLiteral()
		s, ok := lit.(ast.StringLiteralNode)
		if !ok || s.Value != "ab" {
			t.Fatalf("got %#v", lit)
		}
	})

	t.Run("string_short_value", func(t *testing.T) {
		toks := []ast.Token{
			{Type: ast.TokenStringLiteral, Value: `""`},
			{Type: ast.TokenEOF},
		}
		p := setupParser(toks, logger)
		lit := p.parseLiteral()
		s := lit.(ast.StringLiteralNode)
		if s.Value != "" {
			t.Fatalf("got %q", s.Value)
		}
	})

	t.Run("int_literal", func(t *testing.T) {
		toks := []ast.Token{
			{Type: ast.TokenIntLiteral, Value: "7"},
			{Type: ast.TokenEOF},
		}
		p := setupParser(toks, logger)
		lit := p.parseLiteral()
		n := lit.(ast.IntLiteralNode)
		if n.Value != 7 {
			t.Fatal(n.Value)
		}
	})

	t.Run("float_via_int_dot_fraction", func(t *testing.T) {
		toks := []ast.Token{
			{Type: ast.TokenIntLiteral, Value: "3"},
			{Type: ast.TokenDot},
			{Type: ast.TokenFloatLiteral, Value: "14"},
			{Type: ast.TokenEOF},
		}
		p := setupParser(toks, logger)
		lit := p.parseLiteral()
		f := lit.(ast.FloatLiteralNode)
		if f.Value < 3.13 || f.Value > 3.15 {
			t.Fatalf("got %v", f.Value)
		}
	})

	t.Run("bool_true_false", func(t *testing.T) {
		for _, tc := range []struct {
			tok   ast.TokenIdent
			want  bool
			value string
		}{
			{ast.TokenTrue, true, "true"},
			{ast.TokenFalse, false, "false"},
		} {
			toks := []ast.Token{
				{Type: tc.tok, Value: tc.value},
				{Type: ast.TokenEOF},
			}
			p := setupParser(toks, logger)
			b := p.parseLiteral().(ast.BoolLiteralNode)
			if b.Value != tc.want {
				t.Fatal(b.Value)
			}
		}
	})

	t.Run("nil_literal", func(t *testing.T) {
		toks := []ast.Token{
			{Type: ast.TokenNil, Value: "nil"},
			{Type: ast.TokenEOF},
		}
		p := setupParser(toks, logger)
		_, ok := p.parseLiteral().(ast.NilLiteralNode)
		if !ok {
			t.Fatal("expected NilLiteralNode")
		}
	})
}

func TestParseLiteral_array_literal_branches(t *testing.T) {
	logger := ast.SetupTestLogger(nil)

	t.Run("typed_array", func(t *testing.T) {
		toks := []ast.Token{
			{Type: ast.TokenLBracket, Value: "["},
			{Type: ast.TokenIntLiteral, Value: "1"},
			{Type: ast.TokenRBracket, Value: "]"},
			{Type: ast.TokenIdentifier, Value: "int"},
			{Type: ast.TokenEOF},
		}
		p := setupParser(toks, logger)
		arr := p.parseLiteral().(ast.ArrayLiteralNode)
		if arr.Type.Ident != "int" || len(arr.Value) != 1 {
			t.Fatalf("%+v", arr)
		}
	})

	t.Run("implicit_array_type", func(t *testing.T) {
		toks := []ast.Token{
			{Type: ast.TokenLBracket, Value: "["},
			{Type: ast.TokenIntLiteral, Value: "2"},
			{Type: ast.TokenRBracket, Value: "]"},
			{Type: ast.TokenEOF},
		}
		p := setupParser(toks, logger)
		arr := p.parseLiteral().(ast.ArrayLiteralNode)
		if arr.Type.Ident != ast.TypeImplicit {
			t.Fatalf("got %s", arr.Type.Ident)
		}
	})
}

func TestParseLiteral_unexpected_panics(t *testing.T) {
	logger := ast.SetupTestLogger(nil)
	toks := []ast.Token{
		{Type: ast.TokenFunc, Value: "func"},
		{Type: ast.TokenEOF},
	}
	p := setupParser(toks, logger)
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic from FailWithUnexpectedToken")
		}
	}()
	p.parseLiteral()
}
