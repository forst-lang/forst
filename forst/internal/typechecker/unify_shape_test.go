package typechecker

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
)

// TestUnifyShape_namedAliasWithStructLiteralReturn exercises shape unification through CheckTypes
// (register + infer paths used by unify_shape.go).
func TestUnifyShape_namedAliasWithStructLiteralReturn(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

type Point = { x: Int, y: Int }

func origin(): Point {
	return { x: 0, y: 1 }
}

func main() {
	p := origin()
	println("ok")
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	pts := tc.VariableTypes[ast.Identifier("p")]
	if len(pts) != 1 || pts[0].Ident != ast.TypeIdent("Point") {
		t.Fatalf("expected p inferred as Point after origin(), got %#v", pts)
	}
}

// TestUnifyShape_twoShapeLiteralsAssignable checks that compatible anonymous shapes unify for assignment.
func TestUnifyShape_twoShapeLiteralsAssignable(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func main() {
	a := { x: 1, y: 2 }
	b := { x: 3, y: 4 }
	a = b
	println("ok")
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("expected compatible shapes to unify: %v", err)
	}
}

func TestUnifyShape_nestedShapeLiteralReturn(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

type Inner = { n: Int }
type Outer = { inner: Inner }

func mk(): Outer {
	return { inner: { n: 2 } }
}

func main() {
	o := mk()
	println(o.inner.n)
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
}

func TestIsShapeAlias_nominalError(t *testing.T) {
	t.Parallel()
	tc := New(setupTestLogger(nil), false)
	tc.registerType(ast.TypeDefNode{
		Ident: "E",
		Expr: ast.TypeDefErrorExpr{
			Payload: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{
				"m": {Type: &ast.TypeNode{Ident: ast.TypeString}},
			}},
		},
	})
	if !tc.isShapeAlias("E") {
		t.Fatal("nominal error should count as shape alias")
	}
}

func TestIsShapeAlias_unknownType(t *testing.T) {
	t.Parallel()
	tc := New(setupTestLogger(nil), false)
	if tc.isShapeAlias("Missing") {
		t.Fatal("unknown type should not be shape alias")
	}
}

func TestGetShapeFields_fromTypeDefShapeAlias(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := New(log, false)
	tc.registerType(ast.TypeDefNode{
		Ident: "Point",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
					"y": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
				},
			},
		},
	})
	fields, err := tc.getShapeFields(ast.TypeNode{Ident: "Point"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 {
		t.Fatalf("got %d fields", len(fields))
	}
}

func TestGetShapeFields_fromMatchAssertion(t *testing.T) {
	t.Parallel()
	tc := New(setupTestLogger(nil), false)
	shape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	base := ast.TypeShape
	assertion := &ast.AssertionNode{
		BaseType: &base,
		Constraints: []ast.ConstraintNode{{
			Name: ConstraintMatch,
			Args: []ast.ConstraintArgumentNode{{Shape: &shape}},
		}},
	}
	fields, err := tc.getShapeFields(
		ast.TypeNode{Ident: ast.TypeShape, Assertion: assertion},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := fields["name"]; !ok {
		t.Fatalf("got %#v", fields)
	}
}

func TestGetShapeFields_missingFieldsErrors(t *testing.T) {
	t.Parallel()
	tc := New(setupTestLogger(nil), false)
	_, err := tc.getShapeFields(ast.TypeNode{Ident: ast.TypeInt}, nil)
	if err == nil {
		t.Fatal("expected error for non-shape type")
	}
}

func TestValidateShapeFields_compatibleFields(t *testing.T) {
	t.Parallel()
	tc := New(setupTestLogger(nil), false)
	left := map[string]ast.ShapeFieldNode{
		"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
	}
	right := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
		},
	}
	if err := tc.ValidateShapeFields(right, left, "Point"); err != nil {
		t.Fatal(err)
	}
}

func TestValidateShapeFields_unknownFieldErrors(t *testing.T) {
	t.Parallel()
	tc := New(setupTestLogger(nil), false)
	left := map[string]ast.ShapeFieldNode{
		"x": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
	}
	right := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"z": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
		},
	}
	if err := tc.ValidateShapeFields(right, left, "Point"); err == nil {
		t.Fatal("expected unknown field error")
	}
}
