//go:build go1.18

package transformergo

import (
	"forst/internal/ast"
	"forst/internal/typechecker"
	goast "go/ast"
	"testing"
)

func TestCapitalizeFirst(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"foo", "Foo"},
		{"Foo", "Foo"},
		{"f", "F"},
		{"", ""},
		{"ßeta", "SSeta"}, // Unicode: ß uppercases to SS (correct Unicode behavior)
		{"αβγ", "Αβγ"},    // Greek alpha
		{"1foo", "1foo"},  // Non-letter first char
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := capitalizeFirst(tt.input)
			if got != tt.expected {
				t.Errorf("capitalizeFirst(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestShapesMatch_andCompatibility_basicBranches(t *testing.T) {
	tc := typechecker.New(setupTestLogger(nil), false)
	tr := setupTransformer(tc, setupTestLogger(nil))

	s1 := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"a": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	s2 := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"a": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	if !tr.shapesMatch(s1, s2) {
		t.Fatal("expected exact same shapes to match")
	}
	if !tr.shapesCompatibleForExpectedType(s1, s2) {
		t.Fatal("expected exact same shapes to be compatible")
	}

	s3 := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"a": {Type: &ast.TypeNode{Ident: ast.TypeString}},
			"b": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
		},
	}
	if tr.shapesMatch(s1, s3) {
		t.Fatal("expected different field counts not to match")
	}
	if tr.shapesCompatibleForExpectedType(s1, s3) {
		t.Fatal("expected different field counts not to be compatible")
	}
}

func TestResolveFieldToShape_directAndTypedef(t *testing.T) {
	tc := typechecker.New(setupTestLogger(nil), false)
	tr := setupTransformer(tc, setupTestLogger(nil))

	direct := ast.ShapeFieldNode{
		Shape: &ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}}}},
	}
	if got := tr.resolveFieldToShape(direct); got == nil || len(got.Fields) != 1 {
		t.Fatalf("expected direct shape resolution, got %+v", got)
	}

	typeName := ast.TypeIdent("Payload")
	tc.Defs[typeName] = ast.TypeDefNode{
		Ident: typeName,
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{"msg": {Type: &ast.TypeNode{Ident: ast.TypeString}}},
			},
		},
	}
	field := ast.ShapeFieldNode{Type: &ast.TypeNode{Ident: typeName}}
	got := tr.resolveFieldToShape(field)
	if got == nil || got.Fields["msg"].Type == nil || got.Fields["msg"].Type.Ident != ast.TypeString {
		t.Fatalf("expected typedef-backed shape resolution, got %+v", got)
	}
}

func TestTransformShapeType_jsonTagsGatedOnExportStructFields(t *testing.T) {
	shape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"id":        {Type: &ast.TypeNode{Ident: ast.TypeString}},
			"expiresAt": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
		},
		FieldOrder: []string{"id", "expiresAt"},
	}

	t.Run("default omits json tags", func(t *testing.T) {
		tc := typechecker.New(setupTestLogger(nil), false)
		tr := setupTransformer(tc, setupTestLogger(nil))

		expr, err := tr.transformShapeType(shape)
		if err != nil {
			t.Fatal(err)
		}
		st, ok := (*expr).(*goast.StructType)
		if !ok {
			t.Fatalf("expected StructType, got %T", *expr)
		}
		for _, f := range st.Fields.List {
			if f.Tag != nil {
				t.Fatalf("expected no json tag on unexported field %s, got %v", f.Names[0].Name, f.Tag.Value)
			}
			if f.Names[0].Name != "id" && f.Names[0].Name != "expiresAt" {
				t.Fatalf("expected lowercase field names, got %s", f.Names[0].Name)
			}
		}
	})

	t.Run("export emits json tags", func(t *testing.T) {
		tc := typechecker.New(setupTestLogger(nil), false)
		tr := New(tc, setupTestLogger(nil), true)

		expr, err := tr.transformShapeType(shape)
		if err != nil {
			t.Fatal(err)
		}
		st, ok := (*expr).(*goast.StructType)
		if !ok {
			t.Fatalf("expected StructType, got %T", *expr)
		}
		want := map[string]string{
			"Id":        "`json:\"id\"`",
			"ExpiresAt": "`json:\"expiresAt\"`",
		}
		for _, f := range st.Fields.List {
			name := f.Names[0].Name
			tag, ok := want[name]
			if !ok {
				t.Fatalf("unexpected field %s", name)
			}
			if f.Tag == nil || f.Tag.Value != tag {
				t.Fatalf("field %s: got tag %v, want %s", name, f.Tag, tag)
			}
		}
	})
}

func TestTransformMethodOnlyShapeAsInterface_emitsGoInterface(t *testing.T) {
	tc := typechecker.New(setupTestLogger(nil), false)
	tr := setupTransformer(tc, setupTestLogger(nil))

	shape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"info": {
				IsMethod: true,
				MethodParams: []ast.ParamNode{
					ast.SimpleParamNode{
						Ident: ast.Ident{ID: "msg"},
						Type:  ast.TypeNode{Ident: ast.TypeString},
					},
				},
			},
		},
	}
	if !shape.IsMethodOnlyContract() {
		t.Fatal("fixture should be method-only contract")
	}

	expr, err := tr.transformMethodOnlyShapeAsInterface(shape)
	if err != nil {
		t.Fatal(err)
	}
	iface, ok := (*expr).(*goast.InterfaceType)
	if !ok {
		t.Fatalf("expected InterfaceType, got %T", *expr)
	}
	if len(iface.Methods.List) != 1 || iface.Methods.List[0].Names[0].Name != "info" {
		t.Fatalf("interface methods: %+v", iface.Methods.List)
	}
}
