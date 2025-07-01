package transformergo

import (
	"forst/internal/ast"
	goast "go/ast"
	"testing"
)

func TestTransformTypeGuard_Simple(t *testing.T) {
	baseType := ast.TypeInt
	intLit := ast.IntLiteralNode{
		Value: 0,
		Type:  ast.TypeNode{Ident: ast.TypeInt},
	}
	var valueNode ast.ValueNode = intLit
	tg := ast.TypeGuardNode{
		Ident: "IsPositive",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "x"},
			Type:  ast.TypeNode{Ident: ast.TypeInt},
		},
		Body: []ast.Node{
			ast.EnsureNode{
				Variable: ast.VariableNode{
					Ident:        ast.Ident{ID: "x"},
					ExplicitType: ast.TypeNode{Ident: ast.TypeInt},
				},
				Assertion: ast.AssertionNode{
					BaseType: &baseType,
					Constraints: []ast.ConstraintNode{
						{
							Name: "GreaterThan",
							Args: []ast.ConstraintArgumentNode{
								{
									Value: &valueNode,
								},
							},
						},
					},
				},
			},
		},
	}
	log := setupTestLogger()
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	// Register the type guard in the type checker
	if err := tc.CheckTypes([]ast.Node{&tg}); err != nil {
		t.Fatalf("failed to check types: %v", err)
	}

	decl, err := tr.transformTypeGuard(tg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decl == nil || decl.Name == nil || decl.Name.Name == "" {
		t.Errorf("expected function declaration with name, got: %+v", decl)
	}
	if decl.Type == nil || decl.Type.Params == nil || len(decl.Type.Params.List) != 1 {
		t.Errorf("expected one parameter, got: %+v", decl.Type.Params)
	}
	if decl.Type.Results == nil || len(decl.Type.Results.List) != 1 {
		t.Errorf("expected one result, got: %+v", decl.Type.Results)
	}
	if decl.Type.Results.List[0].Type.(*goast.Ident).Name != "bool" {
		t.Errorf("expected result type bool, got: %+v", decl.Type.Results.List[0].Type)
	}
}

func TestTransformTypeGuard_ParamTypes(t *testing.T) {
	baseType := ast.TypeString
	tg := ast.TypeGuardNode{
		Ident: "IsString",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "s"},
			Type:  ast.TypeNode{Ident: ast.TypeString},
		},
		Body: []ast.Node{
			ast.EnsureNode{
				Variable: ast.VariableNode{
					Ident:        ast.Ident{ID: "s"},
					ExplicitType: ast.TypeNode{Ident: ast.TypeString},
				},
				Assertion: ast.AssertionNode{
					BaseType: &baseType,
					Constraints: []ast.ConstraintNode{
						{
							Name: "IsString",
						},
					},
				},
			},
		},
	}
	log := setupTestLogger()
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	// Register the type guard in the type checker
	if err := tc.CheckTypes([]ast.Node{&tg}); err != nil {
		t.Fatalf("failed to check types: %v", err)
	}

	decl, err := tr.transformTypeGuard(tg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decl.Type.Params.List[0].Type.(*goast.Ident).Name != "string" {
		t.Errorf("expected parameter type string, got: %+v", decl.Type.Params.List[0].Type)
	}
}

func TestTransformTypeGuard_DestructuredParamFails(t *testing.T) {
	baseType := ast.TypeInt
	tg := ast.TypeGuardNode{
		Ident: "Destructured",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "x"},
			Type:  ast.TypeNode{Ident: ast.TypeInt},
		},
		Body: []ast.Node{
			ast.EnsureNode{
				Variable: ast.VariableNode{
					Ident:        ast.Ident{ID: "x"},
					ExplicitType: ast.TypeNode{Ident: ast.TypeInt},
				},
				Assertion: ast.AssertionNode{
					BaseType: &baseType,
					Constraints: []ast.ConstraintNode{
						{
							Name: "IsInt",
						},
					},
				},
			},
		},
	}
	log := setupTestLogger()
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	// Register the type guard in the type checker
	if err := tc.CheckTypes([]ast.Node{&tg}); err != nil {
		t.Fatalf("failed to check types: %v", err)
	}

	_, err := tr.transformTypeGuard(tg)
	if err == nil {
		t.Errorf("expected error for DestructuredParamNode, got none")
	}
}

func TestTransformTypeGuard_WithAdditionalParams(t *testing.T) {
	baseType := ast.TypeInt
	other := ast.Ident{ID: "other"}
	otherVar := ast.VariableNode{
		Ident:        other,
		ExplicitType: ast.TypeNode{Ident: ast.TypeInt},
	}
	var valueNode ast.ValueNode = otherVar
	tg := ast.TypeGuardNode{
		Ident: "DivisibleBy",
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "i"},
			Type:  ast.TypeNode{Ident: "Prime"},
		},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: "other"},
				Type:  ast.TypeNode{Ident: ast.TypeInt},
			},
		},
		Body: []ast.Node{
			ast.EnsureNode{
				Variable: ast.VariableNode{
					Ident:        ast.Ident{ID: "i"},
					ExplicitType: ast.TypeNode{Ident: "Prime"},
				},
				Assertion: ast.AssertionNode{
					BaseType: &baseType,
					Constraints: []ast.ConstraintNode{
						{
							Name: "DivisibleBy",
							Args: []ast.ConstraintArgumentNode{
								{
									Value: &valueNode,
								},
							},
						},
					},
				},
			},
		},
	}
	log := setupTestLogger()
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	// Register the type guard in the type checker
	if err := tc.CheckTypes([]ast.Node{&tg}); err != nil {
		t.Fatalf("failed to check types: %v", err)
	}

	decl, err := tr.transformTypeGuard(tg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(decl.Type.Params.List) != 2 {
		t.Errorf("expected two parameters, got: %+v", decl.Type.Params)
	}
	if decl.Type.Params.List[0].Type.(*goast.Ident).Name != "Prime" {
		t.Errorf("expected first parameter type Prime, got: %+v", decl.Type.Params.List[0].Type)
	}
	if decl.Type.Params.List[1].Type.(*goast.Ident).Name != "int" {
		t.Errorf("expected second parameter type int, got: %+v", decl.Type.Params.List[1].Type)
	}
}
