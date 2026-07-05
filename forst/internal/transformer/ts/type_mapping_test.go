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

func TestTypeMapping_GetTypeScriptType_resultPair(t *testing.T) {
	tm := NewTypeMapping()
	r := ast.TypeNode{
		Ident:    ast.TypeResult,
		TypeKind: ast.TypeKindBuiltin,
		TypeParams: []ast.TypeNode{
			ast.NewBuiltinType(ast.TypeInt),
			ast.NewBuiltinType(ast.TypeError),
		},
	}
	got, err := tm.GetTypeScriptType(&r)
	if err != nil {
		t.Fatal(err)
	}
	want := "({ ok: true; value: number } | { ok: false; error: unknown })"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
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

func TestTypeMapping_shapeTypeFieldLines_nestedShapeField(t *testing.T) {
	tm := NewTypeMapping()
	tc := typechecker.New(nil, false)
	tm.SetTypeChecker(tc)
	inner := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"k": {Type: &ast.TypeNode{Ident: ast.TypeString, TypeKind: ast.TypeKindBuiltin}},
		},
	}
	shape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"nested": {Shape: &inner},
		},
	}
	lines, err := tm.shapeTypeFieldLines(shape)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || !strings.Contains(lines[0], "nested") || !strings.Contains(lines[0], "k") {
		t.Fatalf("unexpected lines: %v", lines)
	}
}

func TestTypeMapping_GetTypeScriptType_typeShape_nestedFromAssertion(t *testing.T) {
	tm := NewTypeMapping()
	inner := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"count": {Type: &ast.TypeNode{Ident: ast.TypeInt, TypeKind: ast.TypeKindBuiltin}},
		},
	}
	typ := ast.TypeNode{
		Ident:    ast.TypeShape,
		TypeKind: ast.TypeKindBuiltin,
		Assertion: &ast.AssertionNode{
			Constraints: []ast.ConstraintNode{{
				Name: "Shape",
				Args: []ast.ConstraintArgumentNode{{Shape: &inner}},
			}},
		},
	}
	got, err := tm.GetTypeScriptType(&typ)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  count: number;\n}"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTypeMapping_GetTypeScriptType_goBuiltinIntIdent(t *testing.T) {
	tm := NewTypeMapping()
	got, err := tm.GetTypeScriptType(&ast.TypeNode{Ident: "int", TypeKind: ast.TypeKindBuiltin})
	if err != nil {
		t.Fatal(err)
	}
	if got != "number" {
		t.Fatalf("got %q, want number", got)
	}
}

func TestTypeMapping_GetTypeScriptType_userDefined_whenInDefs(t *testing.T) {
	tc := typechecker.New(nil, false)
	tc.Defs["Widget"] = ast.TypeDefNode{
		Ident: "Widget",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"id": {Type: &ast.TypeNode{Ident: ast.TypeString, TypeKind: ast.TypeKindBuiltin}},
				},
			},
		},
	}
	tm := NewTypeMapping()
	tm.SetTypeChecker(tc)
	got, err := tm.GetTypeScriptType(&ast.TypeNode{
		Ident:    "Widget",
		TypeKind: ast.TypeKindUserDefined,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "Widget" {
		t.Fatalf("got %q, want Widget", got)
	}
}

func TestTypeMapping_GetTypeScriptType_implicit(t *testing.T) {
	tm := NewTypeMapping()
	got, err := tm.GetTypeScriptType(&ast.TypeNode{Ident: ast.TypeImplicit})
	if err != nil {
		t.Fatal(err)
	}
	if got != "unknown" {
		t.Fatalf("got %q, want unknown", got)
	}
}

func TestTypeMapping_GetTypeScriptType_unionAndIntersection(t *testing.T) {
	tc := typechecker.New(nil, false)
	tc.Defs["Tag"] = ast.TypeDefNode{
		Ident: "Tag",
		Expr: ast.TypeDefShapeExpr{Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}},
	}
	tm := NewTypeMapping()
	tm.SetTypeChecker(tc)
	u := ast.NewUnionType(ast.NewBuiltinType(ast.TypeString), ast.NewBuiltinType(ast.TypeInt))
	got, err := tm.GetTypeScriptType(&u)
	if err != nil {
		t.Fatal(err)
	}
	if got != "string | number" {
		t.Fatalf("got %q", got)
	}
	i := ast.NewIntersectionType(ast.NewBuiltinType(ast.TypeString), ast.TypeNode{Ident: "Tag", TypeKind: ast.TypeKindUserDefined})
	got2, err := tm.GetTypeScriptType(&i)
	if err != nil {
		t.Fatal(err)
	}
	if got2 != "string & Tag" {
		t.Fatalf("got %q", got2)
	}
}

func TestGoBuiltinIdentToTypeScript_table(t *testing.T) {
	tests := []struct {
		name  string
		ident ast.TypeIdent
		want  string
		ok    bool
	}{
		{name: "string maps to string", ident: "string", want: "string", ok: true},
		{name: "int64 maps to number", ident: "int64", want: "number", ok: true},
		{name: "bool maps to boolean", ident: "bool", want: "boolean", ok: true},
		{name: "error maps to unknown", ident: "error", want: "unknown", ok: true},
		{name: "go builtin without ts mapping returns false", ident: "complex64", want: "", ok: false},
		{name: "non builtin returns false", ident: "Widget", want: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := goBuiltinIdentToTypeScript(tt.ident)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTypeMapping_shapeFieldToTypeScript_table(t *testing.T) {
	tm := NewTypeMapping()
	tm.SetTypeChecker(typechecker.New(nil, false))

	tests := []struct {
		name  string
		field ast.ShapeFieldNode
		want  string
	}{
		{
			name: "uses explicit field type",
			field: ast.ShapeFieldNode{
				Type: ptrToTypeNode(ast.NewBuiltinType(ast.TypeInt)),
			},
			want: "number",
		},
		{
			name: "uses nested shape",
			field: ast.ShapeFieldNode{
				Shape: &ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"name": {Type: ptrToTypeNode(ast.NewBuiltinType(ast.TypeString))},
					},
				},
			},
			want: "{\n  name: string;\n}",
		},
		{
			name: "falls back to unknown for unsupported assertion",
			field: ast.ShapeFieldNode{
				Assertion: &ast.AssertionNode{},
			},
			want: "any",
		},
		{
			name: "falls back to unknown for empty field",
			field: ast.ShapeFieldNode{},
			want:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tm.shapeFieldToTypeScript(tt.field)
			if err != nil {
				t.Fatalf("shapeFieldToTypeScript() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("shapeFieldToTypeScript() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTypeMapping_GetTypeScriptType_extraCases(t *testing.T) {
	tm := NewTypeMapping()

	tests := []struct {
		name string
		in   ast.TypeNode
		want string
	}{
		{
			name: "array with no element type defaults to any array",
			in: ast.TypeNode{
				Ident:    ast.TypeArray,
				TypeKind: ast.TypeKindBuiltin,
			},
			want: "any[]",
		},
		{
			name: "map with missing value type defaults to any record",
			in: ast.TypeNode{
				Ident: ast.TypeMap,
				TypeParams: []ast.TypeNode{
					ast.NewBuiltinType(ast.TypeString),
				},
				TypeKind: ast.TypeKindBuiltin,
			},
			want: "Record<any, any>",
		},
		{
			name: "pointer with no element type defaults to unknown",
			in: ast.TypeNode{
				Ident:    ast.TypePointer,
				TypeKind: ast.TypeKindBuiltin,
			},
			want: "unknown",
		},
		{
			name: "channel with no element defaults to unknown",
			in: ast.TypeNode{
				Ident:    ast.TypeChannel,
				TypeKind: ast.TypeKindBuiltin,
			},
			want: "unknown",
		},
		{
			name: "union with no type params is never",
			in: ast.TypeNode{
				Ident:    ast.TypeUnion,
				TypeKind: ast.TypeKindBuiltin,
			},
			want: "never",
		},
		{
			name: "intersection with no type params is unknown",
			in: ast.TypeNode{
				Ident:    ast.TypeIntersection,
				TypeKind: ast.TypeKindBuiltin,
			},
			want: "unknown",
		},
		{
			name: "result wraps union and intersection sides with parens",
			in: ast.TypeNode{
				Ident:    ast.TypeResult,
				TypeKind: ast.TypeKindBuiltin,
				TypeParams: []ast.TypeNode{
					ast.NewUnionType(ast.NewBuiltinType(ast.TypeString), ast.NewBuiltinType(ast.TypeInt)),
					ast.NewIntersectionType(ast.NewBuiltinType(ast.TypeString), ast.NewBuiltinType(ast.TypeBool)),
				},
			},
			want: "({ ok: true; value: (string | number) } | { ok: false; error: (string & boolean) })",
		},
		{
			name: "result without success and error params defaults to unknown",
			in: ast.TypeNode{
				Ident:    ast.TypeResult,
				TypeKind: ast.TypeKindBuiltin,
			},
			want: "unknown",
		},
		{
			name: "assertion with base type falls back through base type mapping",
			in: ast.TypeNode{
				Ident: ast.TypeAssertion,
				Assertion: &ast.AssertionNode{
					BaseType: ptrToTypeIdent(ast.TypeString),
				},
			},
			want: "string",
		},
		{
			name: "assertion without inferable info returns unknown",
			in: ast.TypeNode{
				Ident:     ast.TypeAssertion,
				Assertion: &ast.AssertionNode{},
			},
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tm.GetTypeScriptType(&tt.in)
			if err != nil {
				t.Fatalf("GetTypeScriptType() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("GetTypeScriptType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTypeMapping_GetTypeScriptType_assertionUsesTypecheckerInference(t *testing.T) {
	tm := NewTypeMapping()
	tc := typechecker.New(nil, false)
	tm.SetTypeChecker(tc)
	base := ast.TypeString
	in := ast.TypeNode{
		Ident: ast.TypeAssertion,
		Assertion: &ast.AssertionNode{
			BaseType: &base,
		},
	}
	got, err := tm.GetTypeScriptType(&in)
	if err != nil {
		t.Fatal(err)
	}
	if got != "string" {
		t.Fatalf("got %q, want string", got)
	}
}

func TestTypeMapping_shapeHelpers_coverEmptyAndFilledShapes(t *testing.T) {
	tm := NewTypeMapping()
	emptyLines, err := tm.shapeTypeFieldLines(ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}})
	if err != nil {
		t.Fatal(err)
	}
	if emptyLines != nil {
		t.Fatalf("expected nil lines for empty shape, got %v", emptyLines)
	}

	emptyInline, err := tm.shapeToInlineTypeScript(ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}})
	if err != nil {
		t.Fatal(err)
	}
	if emptyInline != "{}" {
		t.Fatalf("got %q, want {}", emptyInline)
	}

	filledInline, err := tm.shapeToInlineTypeScript(ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {Type: ptrToTypeNode(ast.NewBuiltinType(ast.TypeString))},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if filledInline != "{\n  name: string;\n}" {
		t.Fatalf("got %q", filledInline)
	}
}

func TestTypeMapping_GetTypeScriptType_hashBasedWithoutDefinitionsUsesStableAlias(t *testing.T) {
	tm := NewTypeMapping()
	tm.SetTypeChecker(typechecker.New(nil, false))
	got, err := tm.GetTypeScriptType(&ast.TypeNode{
		Ident:    ast.TypeIdent("T_missing_hash"),
		TypeKind: ast.TypeKindHashBased,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got == "" || !strings.HasPrefix(got, "T_") {
		t.Fatalf("got %q, want generated T_* alias", got)
	}
}

func TestTypeMapping_GetTypeScriptType_assertionWithInlineShape(t *testing.T) {
	tm := NewTypeMapping()
	got, err := tm.GetTypeScriptType(&ast.TypeNode{
		Ident: ast.TypeAssertion,
		Assertion: &ast.AssertionNode{
			Constraints: []ast.ConstraintNode{{
				Name: "Shape",
				Args: []ast.ConstraintArgumentNode{{
					Shape: &ast.ShapeNode{
						Fields: map[string]ast.ShapeFieldNode{
							"ok": {Type: ptrToTypeNode(ast.NewBuiltinType(ast.TypeBool))},
						},
					},
				}},
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "{\n  ok: boolean;\n}" {
		t.Fatalf("got %q", got)
	}
}

func ptrToTypeNode(n ast.TypeNode) *ast.TypeNode {
	return &n
}

func ptrToTypeIdent(id ast.TypeIdent) *ast.TypeIdent {
	return &id
}

func TestSortedFieldNames_table(t *testing.T) {
	tests := []struct {
		name   string
		fields map[string]ast.ShapeFieldNode
		want   []string
	}{
		{
			name:   "empty map returns nil",
			fields: map[string]ast.ShapeFieldNode{},
			want:   nil,
		},
		{
			name: "returns stable sorted names",
			fields: map[string]ast.ShapeFieldNode{
				"z": {},
				"a": {},
				"m": {},
			},
			want: []string{"a", "m", "z"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sortedFieldNames(tt.fields)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestExtractInlineShapeFromAssertion_table(t *testing.T) {
	shape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"id": {Type: ptrToTypeNode(ast.NewBuiltinType(ast.TypeInt))},
		},
	}
	tests := []struct {
		name string
		in   *ast.AssertionNode
		want *ast.ShapeNode
	}{
		{
			name: "nil assertion returns nil",
			in:   nil,
			want: nil,
		},
		{
			name: "no matching constraints returns nil",
			in: &ast.AssertionNode{
				Constraints: []ast.ConstraintNode{{Name: "Other"}},
			},
			want: nil,
		},
		{
			name: "shape constraint returns shape",
			in: &ast.AssertionNode{
				Constraints: []ast.ConstraintNode{{
					Name: "Shape",
					Args: []ast.ConstraintArgumentNode{{Shape: &shape}},
				}},
			},
			want: &shape,
		},
		{
			name: "match constraint returns shape",
			in: &ast.AssertionNode{
				Constraints: []ast.ConstraintNode{{
					Name: typechecker.ConstraintMatch,
					Args: []ast.ConstraintArgumentNode{{Shape: &shape}},
				}},
			},
			want: &shape,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractInlineShapeFromAssertion(tt.in)
			if (got == nil) != (tt.want == nil) {
				t.Fatalf("got nil = %v, want nil = %v", got == nil, tt.want == nil)
			}
			if got != nil && len(got.Fields) != len(tt.want.Fields) {
				t.Fatalf("got fields len %d, want %d", len(got.Fields), len(tt.want.Fields))
			}
		})
	}
}
