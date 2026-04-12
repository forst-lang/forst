package transformergo

import (
	"go/format"
	"go/token"
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestTransformAssertionType_valueConstraint_literals(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	cases := []struct {
		name string
		val  ast.ValueNode
		want string // Go type ident name
	}{
		{"int", ast.IntLiteralNode{Value: 7}, "int"},
		{"string", ast.StringLiteralNode{Value: "a"}, "string"},
		{"float", ast.FloatLiteralNode{Value: 1.25}, "float64"},
		{"bool", ast.BoolLiteralNode{Value: false}, "bool"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var v ast.ValueNode = tc.val
			a := &ast.AssertionNode{
				Constraints: []ast.ConstraintNode{
					{Name: ast.ValueConstraint, Args: []ast.ConstraintArgumentNode{{Value: &v}}},
				},
			}
			expr, err := tr.transformAssertionType(a)
			if err != nil {
				t.Fatal(err)
			}
			if expr == nil {
				t.Fatal("nil expr")
			}
			got := goExprString(t, *expr)
			if !strings.Contains(got, tc.want) {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestTransformAssertionType_valueConstraint_noArgs_defaultsString(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)
	a := &ast.AssertionNode{
		Constraints: []ast.ConstraintNode{{Name: ast.ValueConstraint}},
	}
	expr, err := tr.transformAssertionType(a)
	if err != nil {
		t.Fatal(err)
	}
	if goExprString(t, *expr) != "string" {
		t.Fatalf("got %s", goExprString(t, *expr))
	}
}

func TestTransformAssertionType_baseTypeOnly_emitsIdent(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)
	bt := ast.TypeIdent("User")
	a := &ast.AssertionNode{BaseType: &bt, Constraints: nil}
	expr, err := tr.transformAssertionType(a)
	if err != nil {
		t.Fatal(err)
	}
	if goExprString(t, *expr) != "User" {
		t.Fatalf("got %q", goExprString(t, *expr))
	}
}

func TestTransformAssertionType_valueConstraint_referenceNode(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)
	var v ast.ValueNode = ast.ReferenceNode{Value: ast.VariableNode{Ident: ast.Ident{ID: "x"}}}
	a := &ast.AssertionNode{
		Constraints: []ast.ConstraintNode{
			{Name: ast.ValueConstraint, Args: []ast.ConstraintArgumentNode{{Value: &v}}},
		},
	}
	expr, err := tr.transformAssertionType(a)
	if err != nil {
		t.Fatal(err)
	}
	s := goExprString(t, *expr)
	if s != "*string" {
		t.Fatalf("got %q want *string", s)
	}
}

func TestTransformAssertionType_valueConstraint_arrayLiteral_defaultsToString(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)
	var v ast.ValueNode = ast.ArrayLiteralNode{}
	a := &ast.AssertionNode{
		Constraints: []ast.ConstraintNode{
			{Name: ast.ValueConstraint, Args: []ast.ConstraintArgumentNode{{Value: &v}}},
		},
	}
	expr, err := tr.transformAssertionType(a)
	if err != nil {
		t.Fatal(err)
	}
	if goExprString(t, *expr) != "string" {
		t.Fatalf("got %q", goExprString(t, *expr))
	}
}

func goExprString(t *testing.T, expr interface{}) string {
	t.Helper()
	var buf strings.Builder
	if err := format.Node(&buf, token.NewFileSet(), expr); err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(buf.String())
}
