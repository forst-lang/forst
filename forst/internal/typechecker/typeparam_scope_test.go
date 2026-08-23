package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
	"forst/internal/testutil"

	"github.com/sirupsen/logrus"
)

func TestCollectNormalizesGenericSignature(t *testing.T) {
	t.Parallel()
	src := `package main

func identity[T any](x T): T {
	return x
}
`
	p := parser.NewTestParser(src, logrus.New())
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(logrus.New(), false)
	if err := tc.CollectTypes(nodes); err != nil {
		t.Fatal(err)
	}
	sig := tc.Functions[ast.Identifier("identity")]
	if !sig.Parameters[0].Type.IsTypeParam() {
		t.Fatalf("after collect param type = %+v", sig.Parameters[0].Type)
	}
	if err := tc.validateReferencedTypesAfterCollect(); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeGenericSignature_rewritesTypeParams(t *testing.T) {
	t.Parallel()
	fn := ast.FunctionNode{
		Ident: ast.Ident{ID: "identity"},
		TypeParams: []ast.TypeParamDecl{
			{Name: "T", Constraint: &ast.TypeNode{Ident: ast.TypeIdent("any")}},
		},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: "x"},
				Type:  ast.NewUserDefinedType(ast.TypeIdent("T")),
			},
		},
		ReturnTypes: []ast.TypeNode{ast.NewUserDefinedType(ast.TypeIdent("T"))},
	}
	sig := normalizeGenericSignature(fn)
	if !sig.Parameters[0].Type.IsTypeParam() || sig.Parameters[0].Type.Ident != "T" {
		t.Fatalf("param type: %+v", sig.Parameters[0].Type)
	}
	if !sig.ReturnTypes[0].IsTypeParam() {
		t.Fatalf("return type: %+v", sig.ReturnTypes[0])
	}
}

func TestNormalizeGenericSignature_twoGenericsDoNotLeakDefs(t *testing.T) {
	t.Parallel()
	src := `package main

func f[T any](x T): T { return x }
func g[T any](x T): T { return x }

func main() {
	a := f(1)
	b := g("a")
	println(a)
	println(b)
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{FileID: "two_generics.ft"})
}

func TestNormalizeGenericSignature_typeAliasAndGenericCoexist(t *testing.T) {
	t.Parallel()
	src := `package main

type T = Int

func f[T any](x T): T { return x }

func main() {
	n := f(1)
	println(string(n))
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{FileID: "alias_and_generic.ft"})
}

func TestNormalizeGenericSignature_fromParsedAST(t *testing.T) {
	t.Parallel()
	src := `func identity[T any](x T): T { return x }`
	p := parser.NewTestParser("package main\n"+src, logrus.New())
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	fn, ok := nodes[1].(ast.FunctionNode)
	if !ok {
		t.Fatalf("expected FunctionNode, got %T", nodes[1])
	}
	sig := normalizeGenericSignature(fn)
	if !sig.Parameters[0].Type.IsTypeParam() {
		t.Fatalf("parsed param type not normalized: %+v", sig.Parameters[0].Type)
	}
}

func TestInstantiateGenericCall_direct(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	sig := normalizeGenericSignature(ast.FunctionNode{
		Ident:      ast.Ident{ID: "identity"},
		TypeParams: []ast.TypeParamDecl{{Name: "T"}},
		Params: []ast.ParamNode{ast.SimpleParamNode{
			Ident: ast.Ident{ID: "x"},
			Type:  ast.NewUserDefinedType("T"),
		}},
		ReturnTypes: []ast.TypeNode{ast.NewUserDefinedType("T")},
	})
	argTypes := [][]ast.TypeNode{{ast.NewBuiltinType(ast.TypeInt)}}
	inst, err := tc.instantiateGenericCall(sig, argTypes, ast.SourceSpan{})
	if err != nil {
		t.Fatal(err)
	}
	if inst.Parameters[0].Type.Ident != ast.TypeInt {
		t.Fatalf("expected Int param, got %s", inst.Parameters[0].Type.String())
	}
	if inst.ReturnTypes[0].Ident != ast.TypeInt {
		t.Fatalf("expected Int return, got %s", inst.ReturnTypes[0].String())
	}
}

func TestSubstituteType_nestedTypeParams(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	inner := ast.NewTypeParamType("T")
	wrapped := ast.NewArrayType(inner)
	bindings := map[ast.TypeIdent]ast.TypeNode{"T": ast.NewBuiltinType(ast.TypeInt)}
	out := tc.substituteTypeBindings(wrapped, bindings)
	if len(out.TypeParams) != 1 || out.TypeParams[0].Ident != ast.TypeInt {
		t.Fatalf("expected Array(Int), got %s", out.String())
	}
}
