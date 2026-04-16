package typechecker

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestShapeFieldFromTypeDef_andLookupEnsureBaseType(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	scope := NewScope(nil, nil, logrus.New())
	scope.RegisterSymbol("x", []ast.TypeNode{{Ident: ast.TypeInt}}, SymbolVariable)

	ensure := &ast.EnsureNode{
		Variable: ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier("x")}},
	}
	base, err := tc.LookupEnsureBaseType(ensure, scope)
	if err != nil || base == nil || base.Ident != ast.TypeInt {
		t.Fatalf("LookupEnsureBaseType: base=%+v err=%v", base, err)
	}

	tc.Defs["User"] = ast.TypeDefNode{
		Ident: "User",
		Expr: ast.TypeDefShapeExpr{Shape: ast.ShapeNode{
			Fields: map[string]ast.ShapeFieldNode{
				"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
			},
		}},
	}
	if _, ok := tc.ShapeFieldFromTypeDef("User", "name"); !ok {
		t.Fatal("expected ShapeFieldFromTypeDef(User,name) to succeed")
	}
}

func TestNarrowingDisplayHelpers_andGetAliasedTypeName(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	v := ast.VariableNode{Ident: ast.Ident{ID: "Custom"}}
	segs := tc.narrowingPredicateSegmentFromNamedTypeVar(&v)
	if len(segs) == 0 {
		t.Fatal("expected narrowingPredicateSegmentFromNamedTypeVar segment")
	}
	call := formatPredicateCallDisplay(ast.FunctionCallNode{
		Function:  ast.Ident{ID: "Min"},
		Arguments: []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}},
	})
	if call == "" {
		t.Fatal("expected formatPredicateCallDisplay output")
	}
	if got, err := tc.GetAliasedTypeName(ast.TypeNode{Ident: ast.TypeInt}, GetAliasedTypeNameOptions{}); err != nil || got != "int" {
		t.Fatalf("GetAliasedTypeName int: got=%q err=%v", got, err)
	}
	if got, err := tc.GetAliasedTypeName(ast.TypeNode{Ident: ast.TypePointer, TypeParams: []ast.TypeNode{{Ident: ast.TypeString}}}, GetAliasedTypeNameOptions{}); err != nil || got != "*string" {
		t.Fatalf("GetAliasedTypeName pointer: got=%q err=%v", got, err)
	}
}

func TestValidateTypeDefAssertion_andProcessTypeGuardFields(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	subjectType := ast.TypeNode{Ident: ast.TypeInt}
	paramType := ast.TypeNode{Ident: ast.TypeString}
	tc.Defs["HasName"] = ast.TypeGuardNode{
		Ident: "HasName",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "x"},
			Type:  subjectType,
		},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{Ident: ast.Ident{ID: "name"}, Type: paramType},
		},
	}
	baseType := ast.TypeIdent(ast.TypeInt)
	assertion := &ast.AssertionNode{
		BaseType: &baseType,
		Constraints: []ast.ConstraintNode{
			{
				Name: "HasName",
				Args: []ast.ConstraintArgumentNode{
					{Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	}
	if err := tc.validateTypeDefAssertion(assertion, ast.TypeNode{Ident: ast.TypeInt}); err != nil {
		t.Fatalf("validateTypeDefAssertion: %v", err)
	}
	shape := &ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}
	tc.processTypeGuardFields(shape, assertion)
	if _, ok := shape.Fields["name"]; !ok {
		t.Fatal("expected processTypeGuardFields to add field from guard parameter")
	}
}

