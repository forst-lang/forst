package transformer_go

import (
	"forst/internal/ast"
	"forst/internal/typechecker"
	goast "go/ast"
	"testing"
)

func TestTransformTypeGuard_Simple(t *testing.T) {
	tg := ast.TypeGuardNode{
		Ident: "IsPositive",
		SubjectParam: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "x"},
			Type:  ast.TypeNode{Ident: ast.TypeInt},
		},
		Body: []ast.Node{
			ast.ReturnNode{
				Value: ast.BoolLiteralNode{Value: true, Type: ast.TypeNode{Ident: ast.TypeBool}},
			},
		},
	}
	tr := &Transformer{
		TypeChecker: &typechecker.TypeChecker{},
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
	tg := ast.TypeGuardNode{
		Ident: "IsString",
		SubjectParam: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "s"},
			Type:  ast.TypeNode{Ident: ast.TypeString},
		},
		Body: []ast.Node{
			ast.ReturnNode{
				Value: ast.BoolLiteralNode{Value: true, Type: ast.TypeNode{Ident: ast.TypeBool}},
			},
		},
	}
	tr := &Transformer{
		TypeChecker: &typechecker.TypeChecker{},
	}
	decl, err := tr.transformTypeGuard(tg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decl.Type.Params.List[0].Type.(*goast.Ident).Name != "string" {
		t.Errorf("expected parameter type string, got: %+v", decl.Type.Params.List[0].Type)
	}
}

func TestTransformTypeGuard_DestructuredParamPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for DestructuredParamNode, got none")
		}
	}()
	tg := ast.TypeGuardNode{
		Ident: "Destructured",
		SubjectParam: ast.DestructuredParamNode{
			Fields: []string{"a", "b"},
			Type:   ast.TypeNode{Ident: ast.TypeInt},
		},
		Body: []ast.Node{
			ast.ReturnNode{
				Value: ast.BoolLiteralNode{Value: false, Type: ast.TypeNode{Ident: ast.TypeBool}},
			},
		},
	}
	tr := &Transformer{
		TypeChecker: &typechecker.TypeChecker{},
	}
	_, _ = tr.transformTypeGuard(tg)
}

func TestTransformTypeGuard_WithAdditionalParams(t *testing.T) {
	tg := ast.TypeGuardNode{
		Ident: "DivisibleBy",
		SubjectParam: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "i"},
			Type:  ast.TypeNode{Ident: "Prime"},
		},
		AdditionalParams: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: "other"},
				Type:  ast.TypeNode{Ident: ast.TypeInt},
			},
		},
		Body: []ast.Node{
			ast.ReturnNode{
				Value: ast.BinaryExpressionNode{
					Left: ast.BinaryExpressionNode{
						Left: ast.VariableNode{
							Ident: ast.Ident{ID: "i"},
						},
						Operator: ast.TokenEquals,
						Right: ast.VariableNode{
							Ident: ast.Ident{ID: "other"},
						},
					},
					Operator: ast.TokenLogicalOr,
					Right: ast.BinaryExpressionNode{
						Left: ast.VariableNode{
							Ident: ast.Ident{ID: "other"},
						},
						Operator: ast.TokenEquals,
						Right: ast.IntLiteralNode{
							Value: 1,
							Type:  ast.TypeNode{Ident: ast.TypeInt},
						},
					},
				},
			},
		},
	}
	tr := &Transformer{
		TypeChecker: &typechecker.TypeChecker{},
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
