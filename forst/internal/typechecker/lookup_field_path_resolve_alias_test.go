package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/logger"
)

func ptrTypeIdentForAliasTest(id ast.TypeIdent) *ast.TypeIdent {
	return &id
}

func TestResolveTypeAliasChain_unknown_ident_unchanged(t *testing.T) {
	tc := &TypeChecker{
		log:  logger.New(),
		Defs: map[ast.TypeIdent]ast.Node{},
	}
	in := ast.TypeNode{Ident: ast.TypeIdent("NoSuchType")}
	out := tc.resolveTypeAliasChain(in)
	if out.Ident != in.Ident {
		t.Fatalf("expected unchanged ident %q, got %q", in.Ident, out.Ident)
	}
}

func TestResolveTypeAliasChain_follows_assertion_alias_to_builtin(t *testing.T) {
	alias := ast.TypeIdent("MyStrAlias")
	tc := &TypeChecker{
		log: logger.New(),
		Defs: map[ast.TypeIdent]ast.Node{
			alias: ast.TypeDefNode{
				Ident: alias,
				Expr: ast.TypeDefAssertionExpr{
					Assertion: &ast.AssertionNode{BaseType: ptrTypeIdentForAliasTest(ast.TypeString)},
				},
			},
		},
	}
	out := tc.resolveTypeAliasChain(ast.TypeNode{Ident: alias})
	if out.Ident != ast.TypeString {
		t.Fatalf("expected %q, got %q", ast.TypeString, out.Ident)
	}
}

func TestResolveTypeAliasChain_two_hops_to_builtin(t *testing.T) {
	outer := ast.TypeIdent("OuterAlias")
	mid := ast.TypeIdent("MidAlias")
	tc := &TypeChecker{
		log: logger.New(),
		Defs: map[ast.TypeIdent]ast.Node{
			outer: ast.TypeDefNode{
				Ident: outer,
				Expr: ast.TypeDefAssertionExpr{
					Assertion: &ast.AssertionNode{BaseType: ptrTypeIdentForAliasTest(mid)},
				},
			},
			mid: ast.TypeDefNode{
				Ident: mid,
				Expr: ast.TypeDefAssertionExpr{
					Assertion: &ast.AssertionNode{BaseType: ptrTypeIdentForAliasTest(ast.TypeString)},
				},
			},
		},
	}
	out := tc.resolveTypeAliasChain(ast.TypeNode{Ident: outer})
	if out.Ident != ast.TypeString {
		t.Fatalf("expected %q, got %q", ast.TypeString, out.Ident)
	}
}

func TestResolveTypeAliasChain_shape_typedef_stops_at_name(t *testing.T) {
	named := ast.TypeIdent("NamedShape")
	tc := &TypeChecker{
		log: logger.New(),
		Defs: map[ast.TypeIdent]ast.Node{
			named: ast.TypeDefNode{
				Ident: named,
				Expr: ast.TypeDefShapeExpr{
					Shape: ast.ShapeNode{
						Fields: map[string]ast.ShapeFieldNode{
							"x": {Type: &ast.TypeNode{Ident: ast.TypeString}},
						},
					},
				},
			},
		},
	}
	out := tc.resolveTypeAliasChain(ast.TypeNode{Ident: named})
	if out.Ident != named {
		t.Fatalf("expected named shape ident %q, got %q", named, out.Ident)
	}
}

func TestResolveTypeAliasChain_assertion_without_base_returns_current(t *testing.T) {
	alias := ast.TypeIdent("OpaqueAlias")
	tc := &TypeChecker{
		log: logger.New(),
		Defs: map[ast.TypeIdent]ast.Node{
			alias: ast.TypeDefNode{
				Ident: alias,
				Expr: ast.TypeDefAssertionExpr{
					Assertion: &ast.AssertionNode{BaseType: nil},
				},
			},
		},
	}
	out := tc.resolveTypeAliasChain(ast.TypeNode{Ident: alias})
	if out.Ident != alias {
		t.Fatalf("expected %q, got %q", alias, out.Ident)
	}
}

func TestResolveTypeAliasChain_breaks_on_alias_cycle(t *testing.T) {
	a := ast.TypeIdent("CycleA")
	b := ast.TypeIdent("CycleB")
	tc := &TypeChecker{
		log: logger.New(),
		Defs: map[ast.TypeIdent]ast.Node{
			a: ast.TypeDefNode{
				Ident: a,
				Expr: ast.TypeDefAssertionExpr{
					Assertion: &ast.AssertionNode{BaseType: ptrTypeIdentForAliasTest(b)},
				},
			},
			b: ast.TypeDefNode{
				Ident: b,
				Expr: ast.TypeDefAssertionExpr{
					Assertion: &ast.AssertionNode{BaseType: ptrTypeIdentForAliasTest(a)},
				},
			},
		},
	}
	out := tc.resolveTypeAliasChain(ast.TypeNode{Ident: a})
	if out.Ident != a {
		t.Fatalf("expected cycle break to keep start alias %q, got %q", a, out.Ident)
	}
}

func TestLookupFieldPath_resolves_field_type_alias_to_builtin(t *testing.T) {
	root := ast.TypeIdent("RootWithAliasedField")
	innerAlias := ast.TypeIdent("InnerStrAlias")
	tc := &TypeChecker{
		log: logger.New(),
		Defs: map[ast.TypeIdent]ast.Node{
			innerAlias: ast.TypeDefNode{
				Ident: innerAlias,
				Expr: ast.TypeDefAssertionExpr{
					Assertion: &ast.AssertionNode{BaseType: ptrTypeIdentForAliasTest(ast.TypeString)},
				},
			},
			root: ast.TypeDefNode{
				Ident: root,
				Expr: ast.TypeDefShapeExpr{
					Shape: ast.ShapeNode{
						Fields: map[string]ast.ShapeFieldNode{
							"s": {Type: &ast.TypeNode{Ident: innerAlias}},
						},
					},
				},
			},
		},
	}
	out, err := tc.lookupFieldPath(ast.TypeNode{Ident: root}, []string{"s"})
	if err != nil {
		t.Fatalf("lookupFieldPath: %v", err)
	}
	if out.Ident != ast.TypeString {
		t.Fatalf("expected field type resolved to %q, got %q", ast.TypeString, out.Ident)
	}
}
