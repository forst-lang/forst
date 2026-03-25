package transformerts

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker"
)

func TestTypeMapping_GetTypeScriptType_nil(t *testing.T) {
	tm := NewTypeMapping()
	_, err := tm.GetTypeScriptType(nil)
	if err == nil {
		t.Fatal("expected error for nil forstType")
	}
	if !strings.Contains(err.Error(), "forstType required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTypeMapping_GetTypeScriptType_builtins(t *testing.T) {
	tm := NewTypeMapping()
	tests := []struct {
		ident ast.TypeIdent
		want  string
	}{
		{ast.TypeString, "string"},
		{ast.TypeInt, "number"},
		{ast.TypeFloat, "number"},
		{ast.TypeBool, "boolean"},
		{ast.TypeVoid, "void"},
		{ast.TypeShape, "object"},
		{ast.TypeObject, "object"},
		{ast.TypeArray, "any[]"},
		{ast.TypeMap, "Record<any, any>"},
	}
	for _, tt := range tests {
		t.Run(string(tt.ident), func(t *testing.T) {
			got, err := tm.GetTypeScriptType(&ast.TypeNode{Ident: tt.ident, TypeKind: ast.TypeKindBuiltin})
			if err != nil {
				t.Fatalf("GetTypeScriptType: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTypeMapping_GetTypeScriptType_arrayWithElementType(t *testing.T) {
	tm := NewTypeMapping()
	got, err := tm.GetTypeScriptType(&ast.TypeNode{
		Ident:      ast.TypeArray,
		TypeKind:   ast.TypeKindBuiltin,
		TypeParams: []ast.TypeNode{ast.NewBuiltinType(ast.TypeString)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "string[]" {
		t.Fatalf("got %q, want string[]", got)
	}
}

func TestTypeMapping_GetTypeScriptType_mapWithKeyValue(t *testing.T) {
	tm := NewTypeMapping()
	m := ast.NewMapType(ast.NewBuiltinType(ast.TypeString), ast.NewBuiltinType(ast.TypeInt))
	got, err := tm.GetTypeScriptType(&m)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Record<string, number>" {
		t.Fatalf("got %q, want Record<string, number>", got)
	}
}

func TestTypeMapping_GetTypeScriptType_pointerToNumber(t *testing.T) {
	tm := NewTypeMapping()
	p := ast.NewPointerType(ast.NewBuiltinType(ast.TypeInt))
	got, err := tm.GetTypeScriptType(&p)
	if err != nil {
		t.Fatal(err)
	}
	if got != "(number) | null" {
		t.Fatalf("got %q, want (number) | null", got)
	}
}

func TestTypeMapping_GetTypeScriptType_errorBuiltin(t *testing.T) {
	tm := NewTypeMapping()
	got, err := tm.GetTypeScriptType(&ast.TypeNode{Ident: ast.TypeError, TypeKind: ast.TypeKindBuiltin})
	if err != nil {
		t.Fatal(err)
	}
	if got != "unknown" {
		t.Fatalf("got %q, want unknown", got)
	}
}

func TestTypeMapping_arrayTypeScript_parenthesizesUnions(t *testing.T) {
	if got := arrayTypeScript("string | number"); got != "(string | number)[]" {
		t.Fatalf("got %q", got)
	}
	if got := arrayTypeScript("string"); got != "string[]" {
		t.Fatalf("got %q", got)
	}
}

func TestTypeMapping_GetTypeScriptType_userTypeOverridesBuiltinName(t *testing.T) {
	tm := NewTypeMapping()
	// User map is keyed by string(Ident); overriding the string form of a builtin blocks the builtin branch.
	tm.AddUserType(string(ast.TypeString), "CustomString")

	got, err := tm.GetTypeScriptType(&ast.TypeNode{Ident: ast.TypeString, TypeKind: ast.TypeKindBuiltin})
	if err != nil {
		t.Fatal(err)
	}
	if got != "CustomString" {
		t.Fatalf("got %q, want CustomString", got)
	}
}

func TestTypeMapping_GetTypeScriptType_unknownIdentReturnsAny(t *testing.T) {
	tm := NewTypeMapping()
	got, err := tm.GetTypeScriptType(&ast.TypeNode{
		Ident:    "SomeUnknownNamedType",
		TypeKind: ast.TypeKindUserDefined,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "any" {
		t.Fatalf("got %q, want any", got)
	}
}

func TestTypeMapping_GetTypeScriptType_hashBased_usesTypecheckerAliasWhenInDefs(t *testing.T) {
	// When the hash ident is registered in Defs, GetAliasedTypeName returns that name
	// (see typechecker.GetAliasedTypeName), so the mapper emits the stable type ident.
	hashID := ast.TypeIdent("T_tsTestHash1")
	tc := typechecker.New(nil, false)
	tc.Defs[hashID] = ast.TypeDefNode{
		Ident: hashID,
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"label": {Type: &ast.TypeNode{Ident: ast.TypeString, TypeKind: ast.TypeKindBuiltin}},
					"count": {Type: &ast.TypeNode{Ident: ast.TypeInt, TypeKind: ast.TypeKindBuiltin}},
				},
			},
		},
	}

	tm := NewTypeMapping()
	tm.SetTypeChecker(tc)

	out, err := tm.GetTypeScriptType(&ast.TypeNode{
		Ident:    hashID,
		TypeKind: ast.TypeKindHashBased,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != string(hashID) {
		t.Fatalf("got %q, want %q", out, hashID)
	}
}
