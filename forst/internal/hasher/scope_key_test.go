package hasher

import (
	"testing"

	"forst/internal/ast"
)

func TestHashScopeKey_functionExcludesBody(t *testing.T) {
	t.Parallel()
	body200 := make([]ast.Node, 200)
	for i := range body200 {
		body200[i] = ast.IntLiteralNode{Value: int64(i)}
	}
	fn := ast.FunctionNode{
		Ident: ast.Ident{ID: "f"},
		Body:  body200,
	}
	fnEmpty := fn
	fnEmpty.Body = nil

	h := New()
	scopeHash, err := h.HashScopeKey(fn)
	if err != nil {
		t.Fatal(err)
	}
	scopeEmpty, err := h.HashScopeKey(fnEmpty)
	if err != nil {
		t.Fatal(err)
	}
	if scopeHash != scopeEmpty {
		t.Fatalf("scope key should ignore body: %x vs %x", scopeHash, scopeEmpty)
	}

	full, err := h.HashNode(fn)
	if err != nil {
		t.Fatal(err)
	}
	fullEmpty, err := h.HashNode(fnEmpty)
	if err != nil {
		t.Fatal(err)
	}
	if full == fullEmpty {
		t.Fatal("full HashNode must include body")
	}
}

func TestHashScopeKey_distinctEnsureSubjects(t *testing.T) {
	t.Parallel()
	assertion := ast.AssertionNode{Constraints: []ast.ConstraintNode{{Name: "Min"}}}
	e1 := ast.EnsureNode{
		Variable:  ast.VariableNode{Ident: ast.Ident{ID: "a"}},
		Assertion: assertion,
	}
	e2 := ast.EnsureNode{
		Variable:  ast.VariableNode{Ident: ast.Ident{ID: "b"}},
		Assertion: assertion,
	}
	h := New()
	h1, err := h.HashScopeKey(e1)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := h.HashScopeKey(e2)
	if err != nil {
		t.Fatal(err)
	}
	if h1 == h2 {
		t.Fatalf("distinct ensure subjects must not share scope key: %x", h1)
	}
}

func TestHashScopeKey_typeGuardExcludesBody(t *testing.T) {
	t.Parallel()
	subject := ast.SimpleParamNode{
		Ident: ast.Ident{ID: "x"},
		Type:  ast.TypeNode{Ident: ast.TypeString},
	}
	body := make([]ast.Node, 100)
	for i := range body {
		body[i] = ast.ReturnNode{Values: []ast.ExpressionNode{ast.BoolLiteralNode{Value: true}}}
	}
	tg := ast.TypeGuardNode{
		Ident:   "G",
		Subject: subject,
		Body:    body,
	}
	tgEmpty := tg
	tgEmpty.Body = nil

	h := New()
	sk, err := h.HashScopeKey(tg)
	if err != nil {
		t.Fatal(err)
	}
	skEmpty, err := h.HashScopeKey(tgEmpty)
	if err != nil {
		t.Fatal(err)
	}
	if sk != skEmpty {
		t.Fatalf("scope key should ignore type guard body")
	}
}
