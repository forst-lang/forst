package parser

import (
	"testing"

	"forst/internal/ast"
)

func TestTokenCanStartAssertionBaseType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tok  ast.Token
		want bool
	}{
		{name: "identifier", tok: ast.Token{Type: ast.TokenIdentifier, Value: "User"}, want: true},
		{name: "string keyword", tok: ast.Token{Type: ast.TokenString, Value: "String"}, want: true},
		{name: "int keyword", tok: ast.Token{Type: ast.TokenInt, Value: "Int"}, want: true},
		{name: "float keyword", tok: ast.Token{Type: ast.TokenFloat, Value: "Float"}, want: true},
		{name: "bool keyword", tok: ast.Token{Type: ast.TokenBool, Value: "Bool"}, want: true},
		{name: "left brace", tok: ast.Token{Type: ast.TokenLBrace, Value: "{"}, want: false},
		{name: "plus", tok: ast.Token{Type: ast.TokenPlus, Value: "+"}, want: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tokenCanStartAssertionBaseType(tc.tok); got != tc.want {
				t.Fatalf("tokenCanStartAssertionBaseType(%s) = %v, want %v", tc.tok.Type, got, tc.want)
			}
		})
	}
}

func TestParseAssertionChain(t *testing.T) {
	t.Parallel()

	t.Run("constraint_only_allowed_without_base_type", func(t *testing.T) {
		t.Parallel()
		p := NewTestParser(`Min(1)`, ast.SetupTestLogger(nil))
		assertion := p.parseAssertionChain(false)
		if assertion.BaseType != nil {
			t.Fatalf("expected nil base type, got %v", *assertion.BaseType)
		}
		if len(assertion.Constraints) != 1 || assertion.Constraints[0].Name != "Min" {
			t.Fatalf("unexpected constraints: %+v", assertion.Constraints)
		}
	})

	t.Run("constraint_only_rejected_when_base_type_required", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected parse panic for missing base type")
			}
		}()
		p := NewTestParser(`Min(1)`, ast.SetupTestLogger(nil))
		_ = p.parseAssertionChain(true)
	})

	t.Run("qualified_type_with_constraint_chain", func(t *testing.T) {
		t.Parallel()
		p := NewTestParser(`pkg.User.Min(1)`, ast.SetupTestLogger(nil))
		assertion := p.parseAssertionChain(true)
		if assertion.BaseType == nil || *assertion.BaseType != "pkg.User" {
			t.Fatalf("base type = %v, want pkg.User", assertion.BaseType)
		}
		if len(assertion.Constraints) != 1 || assertion.Constraints[0].Name != "Min" {
			t.Fatalf("unexpected constraints: %+v", assertion.Constraints)
		}
	})

	t.Run("builtin_base_type_with_constraint_chain", func(t *testing.T) {
		t.Parallel()
		p := NewTestParser(`String.Min(1)`, ast.SetupTestLogger(nil))
		assertion := p.parseAssertionChain(true)
		if assertion.BaseType == nil || *assertion.BaseType != ast.TypeString {
			t.Fatalf("base type = %v, want String", assertion.BaseType)
		}
		if len(assertion.Constraints) != 1 || assertion.Constraints[0].Name != "Min" {
			t.Fatalf("unexpected constraints: %+v", assertion.Constraints)
		}
	})

	t.Run("constraint_with_shape_argument", func(t *testing.T) {
		t.Parallel()
		p := NewTestParser(`Input({ x: Int })`, ast.SetupTestLogger(nil))
		assertion := p.parseAssertionChain(false)
		if len(assertion.Constraints) != 1 || assertion.Constraints[0].Args[0].Shape == nil {
			t.Fatalf("unexpected constraint args: %+v", assertion.Constraints)
		}
	})

	t.Run("multiple_constraints_in_chain", func(t *testing.T) {
		t.Parallel()
		p := NewTestParser(`String.Min(1).Max(10)`, ast.SetupTestLogger(nil))
		assertion := p.parseAssertionChain(true)
		if len(assertion.Constraints) != 2 {
			t.Fatalf("constraints = %+v", assertion.Constraints)
		}
	})
}
