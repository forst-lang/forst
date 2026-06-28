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
