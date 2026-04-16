package transformergo

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
	"forst/internal/typechecker"
)

func TestGetExpectedTypeForShape_contextResolutionPaths(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	shape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	tc.Defs["User"] = ast.TypeDefNode{
		Ident: "User",
		Expr:  ast.TypeDefShapeExpr{Shape: shape},
	}

	// Explicit expected type branch.
	if got := tr.getExpectedTypeForShape(&shape, &ShapeContext{
		ExpectedType: &ast.TypeNode{Ident: "User"},
	}); got == nil || got.Ident != "User" {
		t.Fatalf("expected explicit expected type User, got %+v", got)
	}

	// VariableName branch.
	tc.VariableTypes["u"] = []ast.TypeNode{{Ident: "User"}}
	if got := tr.getExpectedTypeForShape(&shape, &ShapeContext{
		VariableName: "u",
	}); got == nil || got.Ident != "User" {
		t.Fatalf("expected variable inferred type User, got %+v", got)
	}

	// Function parameter branch.
	tc.Functions["mk"] = typechecker.FunctionSignature{
		Ident: ast.Ident{ID: "mk"},
		Parameters: []typechecker.ParameterSignature{
			{Ident: ast.Ident{ID: "u"}, Type: ast.TypeNode{Ident: "User"}},
		},
	}
	if got := tr.getExpectedTypeForShape(&shape, &ShapeContext{
		FunctionName:   "mk",
		ParameterIndex: 0,
	}); got == nil || got.Ident != "User" {
		t.Fatalf("expected function parameter type User, got %+v", got)
	}

	// Function return branch.
	tc.Functions["ret"] = typechecker.FunctionSignature{
		Ident:       ast.Ident{ID: "ret"},
		ReturnTypes: []ast.TypeNode{{Ident: "User"}},
	}
	if got := tr.getExpectedTypeForShape(&shape, &ShapeContext{
		FunctionName: "ret",
		ReturnIndex:  0,
	}); got == nil || got.Ident != "User" {
		t.Fatalf("expected function return type User, got %+v", got)
	}
}

func TestTransformTypeGuard_directFromParsedNodes(t *testing.T) {
	t.Parallel()
	src := `package main

type Password = String

is (password Password) Strong(min Int) {
	if password is Min(min) {
		println("ok")
	} else {
		println("no")
	}
}
`
	log := ast.SetupTestLogger(nil)
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := typechecker.New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	tr := New(tc, log)

	var guard ast.TypeGuardNode
	found := false
	for _, n := range nodes {
		if g, ok := n.(ast.TypeGuardNode); ok {
			guard = g
			found = true
			break
		}
		if gp, ok := n.(*ast.TypeGuardNode); ok && gp != nil {
			guard = *gp
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected parsed type guard")
	}

	decl, err := tr.transformTypeGuard(guard)
	if err == nil {
		t.Fatalf("expected known if-is codegen limitation error, got decl=%#v", decl)
	}
}

