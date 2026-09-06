package gointerop_test

import (
	"go/types"
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker/gointerop"
)

func TestMapGoType_mapArrayChan(t *testing.T) {
	t.Parallel()
	m := types.NewMap(types.Typ[types.String], types.Typ[types.Int])
	got, ok := gointerop.TypeToForstType(m)
	if !ok || got.Ident != ast.TypeMap || len(got.TypeParams) != 2 {
		t.Fatalf("map: got ok=%v %#v", ok, got)
	}

	arr := types.NewArray(types.Typ[types.Int], 4)
	got, ok = gointerop.TypeToForstType(arr)
	if !ok || got.Ident != ast.TypeArray || got.ArrayLen == nil || *got.ArrayLen != 4 {
		t.Fatalf("array: got ok=%v %#v", ok, got)
	}

	ch := types.NewChan(types.SendRecv, types.Typ[types.String])
	got, ok = gointerop.TypeToForstType(ch)
	if !ok || got.Ident != ast.TypeChannel {
		t.Fatalf("chan: got ok=%v %#v", ok, got)
	}
}

func TestMapGoType_typeAliasToMap(t *testing.T) {
	t.Parallel()
	obj := types.NewTypeName(0, nil, "M", nil)
	underlying := types.NewMap(types.Typ[types.String], types.Typ[types.Int])
	alias := types.NewAlias(obj, underlying)
	got, ok := gointerop.TypeToForstType(alias)
	if !ok || got.Ident != ast.TypeMap {
		t.Fatalf("alias to map: got ok=%v %#v", ok, got)
	}
}

func TestMapGoType_distinctNumericKinds(t *testing.T) {
	t.Parallel()
	u64, ok := gointerop.TypeToForstType(types.Typ[types.Uint64])
	if !ok || u64.Ident != ast.TypeIdent("uint64") {
		t.Fatalf("uint64: got ok=%v %#v", ok, u64)
	}
	i64, ok := gointerop.TypeToForstType(types.Typ[types.Int64])
	if !ok || i64.Ident != ast.TypeIdent("int64") {
		t.Fatalf("int64: got ok=%v %#v", ok, i64)
	}
	if u64.Ident == ast.TypeInt || i64.Ident == ast.TypeInt {
		t.Fatal("fixed-width ints must not collapse to TYPE_INT without host")
	}
}

func TestMapGoType_unnamedStruct_mapsToShape(t *testing.T) {
	t.Parallel()
	st := types.NewStruct([]*types.Var{
		types.NewField(0, nil, "N", types.Typ[types.Int], false),
	}, nil)
	got, ok := gointerop.TypeToForstType(st)
	if !ok || got.Ident != ast.TypeShape || got.Assertion == nil {
		t.Fatalf("want shape, got ok=%v %#v", ok, got)
	}
	fields, ok := shapeFieldsFromMappedType(got)
	if !ok || len(fields) != 1 {
		t.Fatalf("fields: ok=%v len=%d", ok, len(fields))
	}
	if ft, ok := fields["n"]; !ok || ft.Type == nil || ft.Type.Ident != ast.TypeInt {
		t.Fatalf("field n: %#v ok=%v", fields["n"], ok)
	}
}

func TestMapGoType_unexportedStructField_staysImplicit(t *testing.T) {
	t.Parallel()
	st := types.NewStruct([]*types.Var{
		types.NewField(0, nil, "n", types.Typ[types.Int], false),
	}, nil)
	got, ok := gointerop.TypeToForstType(st)
	if !ok || got.Ident != ast.TypeImplicit {
		t.Fatalf("want implicit, got ok=%v %#v", ok, got)
	}
}

func shapeFieldsFromMappedType(tn ast.TypeNode) (map[string]ast.ShapeFieldNode, bool) {
	if tn.Assertion == nil {
		return nil, false
	}
	for _, c := range tn.Assertion.Constraints {
		if c.Name == "Match" && len(c.Args) > 0 && c.Args[0].Shape != nil {
			return c.Args[0].Shape.Fields, true
		}
	}
	return nil, false
}

func TestForstAssignableToGoType_uintRejectsNegativeInt(t *testing.T) {
	t.Parallel()
	host := numericHost{}
	f := ast.TypeNode{Ident: ast.TypeIdent("int64")}
	if gointerop.ForstAssignableToGoType(host, f, types.Typ[types.Uint]) {
		t.Fatal("Forst int64 must not assign to Go uint without conversion")
	}
}

type numericHost struct{}

func (numericHost) ForstTypeForGoType(_ types.Type) (ast.TypeNode, bool) {
	return ast.TypeNode{}, false
}
func (numericHost) IsTypeCompatible(_, _ ast.TypeNode) bool { return false }
func (numericHost) GoTypeForForstType(f ast.TypeNode) types.Type {
	switch f.Ident {
	case ast.TypeInt:
		return types.Typ[types.Int]
	case ast.TypeIdent("int64"):
		return types.Typ[types.Int64]
	}
	return nil
}
