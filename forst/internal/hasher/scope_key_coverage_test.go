package hasher

import (
	"testing"

	"forst/internal/ast"
)

func TestHashScopeKey_allScopeOwningCases(t *testing.T) {
	t.Parallel()
	h := New()
	assertion := ast.AssertionNode{Constraints: []ast.ConstraintNode{{Name: "Min"}}}
	errMsg := ast.EnsureErrorCall{ErrorType: "E", ErrorArgs: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}}}
	var errNode ast.EnsureErrorNode = errMsg
	receiver := ast.SimpleParamNode{Ident: ast.Ident{ID: "self"}, Type: ast.TypeNode{Ident: ast.TypeString}}
	retType := ast.TypeNode{Ident: ast.TypeInt}

	fn := ast.FunctionNode{
		Ident:       ast.Ident{ID: "f"},
		Receiver:    &receiver,
		Params:      []ast.ParamNode{ast.SimpleParamNode{Ident: ast.Ident{ID: "x"}, Type: ast.TypeNode{Ident: ast.TypeInt}}},
		ReturnTypes: []ast.TypeNode{retType},
		Body:        []ast.Node{ast.ReturnNode{Values: []ast.ExpressionNode{ast.IntLiteralNode{Value: 0}}}},
	}
	fnPtr := &fn
	var nilFn *ast.FunctionNode

	ensure := ast.EnsureNode{
		Variable:  ast.VariableNode{Ident: ast.Ident{ID: "v"}},
		Assertion: assertion,
		Error:     &errNode,
	}
	ensurePtr := &ensure
	var nilEnsure *ast.EnsureNode

	block := ast.EnsureBlockNode{Body: []ast.Node{ast.IntLiteralNode{Value: 1}}}
	blockPtr := &block
	var nilBlock *ast.EnsureBlockNode

	tg := ast.TypeGuardNode{
		Ident: "G",
		Subject: ast.DestructuredParamNode{
			Fields: []string{"z", "a"},
			Type:   ast.TypeNode{Ident: ast.TypeInt},
		},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{Ident: ast.Ident{ID: "b"}, Type: ast.TypeNode{Ident: ast.TypeInt}},
			ast.DestructuredParamNode{Fields: []string{"c"}, Type: ast.TypeNode{Ident: ast.TypeString}},
		},
		Body: []ast.Node{ast.ReturnNode{Values: []ast.ExpressionNode{ast.BoolLiteralNode{Value: true}}}},
	}
	tgPtr := &tg
	var nilTG *ast.TypeGuardNode

	lit := ast.FunctionLiteralNode{
		Params:      []ast.ParamNode{ast.SimpleParamNode{Ident: ast.Ident{ID: "y"}, Type: ast.TypeNode{Ident: ast.TypeBool}}},
		ReturnTypes: []ast.TypeNode{ast.TypeNode{Ident: ast.TypeBool}},
		Body:        []ast.Node{ast.ReturnNode{Values: []ast.ExpressionNode{ast.BoolLiteralNode{Value: false}}}},
	}
	litPtr := &lit
	var nilLit *ast.FunctionLiteralNode

	cases := []struct {
		name string
		node ast.Node
	}{
		{"FunctionNode_value", fn},
		{"FunctionNode_ptr", fnPtr},
		{"FunctionNode_nil_ptr", nilFn},
		{"EnsureNode_value", ensure},
		{"EnsureNode_ptr", ensurePtr},
		{"EnsureNode_nil_ptr", nilEnsure},
		{"EnsureBlockNode_value", block},
		{"EnsureBlockNode_ptr", blockPtr},
		{"EnsureBlockNode_nil_ptr", nilBlock},
		{"TypeGuardNode_value", tg},
		{"TypeGuardNode_ptr", tgPtr},
		{"TypeGuardNode_nil_ptr", nilTG},
		{"FunctionLiteralNode_value", lit},
		{"FunctionLiteralNode_ptr", litPtr},
		{"FunctionLiteralNode_nil_ptr", nilLit},
		{"default_VariableNode", ast.VariableNode{Ident: ast.Ident{ID: "x"}}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := h.HashScopeKey(tc.node)
			if err != nil {
				t.Fatal(err)
			}
			got2, err := h.HashScopeKey(tc.node)
			if err != nil {
				t.Fatal(err)
			}
			if got != got2 {
				t.Fatalf("not deterministic: %x vs %x", got, got2)
			}
			if tc.node != nil && got == 0 && got != NodeHash(NilHash) {
				t.Fatalf("unexpected zero hash for %s", tc.name)
			}
		})
	}

	fnOther := fn
	fnOther.Ident = ast.Ident{ID: "g"}
	hOther, err := h.HashScopeKey(fnOther)
	if err != nil {
		t.Fatal(err)
	}
	hFn, err := h.HashScopeKey(fn)
	if err != nil {
		t.Fatal(err)
	}
	if hOther == hFn {
		t.Fatal("distinct functions must not share scope key")
	}
}

func TestHashScopeKeyDisambiguated_mixesIdentity(t *testing.T) {
	t.Parallel()
	h := New()
	fn := &ast.FunctionNode{Ident: ast.Ident{ID: "f"}}
	base, err := h.HashScopeKey(fn)
	if err != nil {
		t.Fatal(err)
	}
	dis, err := h.HashScopeKeyDisambiguated(fn, base)
	if err != nil {
		t.Fatal(err)
	}
	if dis == base {
		t.Fatal("disambiguated key should differ from base when node has identity")
	}
	dis2, err := h.HashScopeKeyDisambiguated(fn, base)
	if err != nil {
		t.Fatal(err)
	}
	if dis != dis2 {
		t.Fatal("disambiguated key not deterministic")
	}

	fn2 := &ast.FunctionNode{Ident: ast.Ident{ID: "f"}}
	disOther, err := h.HashScopeKeyDisambiguated(fn2, base)
	if err != nil {
		t.Fatal(err)
	}
	if dis == disOther {
		t.Fatal("distinct nodes with same base should disambiguate differently")
	}
}
