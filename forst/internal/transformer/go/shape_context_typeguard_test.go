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

	t.Run("baseTypeOnShape", func(t *testing.T) {
		t.Parallel()
		bt := ast.TypeIdent("User")
		s := ast.ShapeNode{
			BaseType: &bt,
			Fields: map[string]ast.ShapeFieldNode{
				"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
			},
		}
		got := tr.getExpectedTypeForShape(&s, &ShapeContext{
			ExpectedType: &ast.TypeNode{Ident: "Other"},
		})
		if got == nil || got.Ident != "User" {
			t.Fatalf("want BaseType User (ignore context), got %+v", got)
		}
	})

	t.Run("hashExpectedTypeResolvesToNamed", func(t *testing.T) {
		t.Parallel()
		hashID := ast.TypeIdent("T_covhash01")
		tcLocal := setupTypeChecker(setupTestLogger(nil))
		trLocal := setupTransformer(tcLocal, setupTestLogger(nil))
		sh := ast.ShapeNode{
			Fields: map[string]ast.ShapeFieldNode{
				"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
			},
		}
		tcLocal.Defs["User"] = ast.TypeDefNode{
			Ident: "User",
			Expr:  ast.TypeDefShapeExpr{Shape: sh},
		}
		tcLocal.Defs[hashID] = ast.TypeDefNode{
			Ident: hashID,
			Expr:  ast.TypeDefShapeExpr{Shape: sh},
		}
		got := trLocal.getExpectedTypeForShape(&sh, &ShapeContext{
			ExpectedType: &ast.TypeNode{Ident: hashID, TypeKind: ast.TypeKindHashBased},
		})
		if got == nil || got.Ident != "User" {
			t.Fatalf("want hash expected resolved to named User, got %+v", got)
		}
	})

	t.Run("expectedTypeIncompatibleThenNilWhenNoStructuralMatch", func(t *testing.T) {
		t.Parallel()
		tcLocal := setupTypeChecker(setupTestLogger(nil))
		trLocal := setupTransformer(tcLocal, setupTestLogger(nil))
		userShape := ast.ShapeNode{
			Fields: map[string]ast.ShapeFieldNode{
				"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
			},
		}
		tcLocal.Defs["User"] = ast.TypeDefNode{
			Ident: "User",
			Expr:  ast.TypeDefShapeExpr{Shape: userShape},
		}
		wrongLiteral := ast.ShapeNode{
			Fields: map[string]ast.ShapeFieldNode{
				"not_name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
			},
		}
		got := trLocal.getExpectedTypeForShape(&wrongLiteral, &ShapeContext{
			ExpectedType: &ast.TypeNode{Ident: "User"},
		})
		if got != nil {
			t.Fatalf("want nil when literal incompatible with expected and no other struct match, got %+v", got)
		}
	})

	t.Run("noDefsReturnsNil", func(t *testing.T) {
		t.Parallel()
		tcLocal := setupTypeChecker(setupTestLogger(nil))
		trLocal := setupTransformer(tcLocal, setupTestLogger(nil))
		orphan := ast.ShapeNode{
			Fields: map[string]ast.ShapeFieldNode{
				"z": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
			},
		}
		if got := trLocal.getExpectedTypeForShape(&orphan, nil); got != nil {
			t.Fatalf("want nil with empty Defs, got %+v", got)
		}
	})

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

// Regression: parameter type TYPE_ASSERTION + BaseType must infer concrete named type (see getExpectedTypeForShape assertion branch).
// Literal must not structurally match User (else findExistingTypeForShape wins before the param block). Extra field → shapesMatch fails; ValidateShapeFields still ok.
func TestGetExpectedTypeForShape_functionParameterAssertionInfersNamedType(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	userShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	tc.Defs["User"] = ast.TypeDefNode{
		Ident: "User",
		Expr:  ast.TypeDefShapeExpr{Shape: userShape},
	}

	literal := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
			"extra": {
				Type: &ast.TypeNode{Ident: ast.TypeString},
			},
		},
	}

	userIdent := ast.TypeIdent("User")
	paramType := ast.TypeNode{
		Ident: ast.TypeAssertion,
		Assertion: &ast.AssertionNode{
			BaseType: &userIdent,
		},
	}
	tc.Functions["f"] = typechecker.FunctionSignature{
		Ident: ast.Ident{ID: "f"},
		Parameters: []typechecker.ParameterSignature{
			{Ident: ast.Ident{ID: "x"}, Type: paramType},
		},
	}

	got := tr.getExpectedTypeForShape(&literal, &ShapeContext{
		FunctionName:   "f",
		ParameterIndex: 0,
		ReturnIndex:    -1,
	})
	if got == nil || got.Ident != "User" {
		t.Fatalf("want assertion param inferred as User, got %+v", got)
	}
}

func TestGetExpectedTypeForShape_variableTypeIncompatibleReturnsNil(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	userShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	tc.Defs["User"] = ast.TypeDefNode{
		Ident: "User",
		Expr:  ast.TypeDefShapeExpr{Shape: userShape},
	}
	tc.VariableTypes["u"] = []ast.TypeNode{{Ident: "User"}}

	wrong := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"not_name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	got := tr.getExpectedTypeForShape(&wrong, &ShapeContext{
		VariableName:   "u",
		ParameterIndex: -1,
		ReturnIndex:    -1,
	})
	if got != nil {
		t.Fatalf("want nil when variable type cannot validate literal, got %+v", got)
	}
}

func TestGetExpectedTypeForShape_nonAssertionParameterSupersetLiteral(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	userShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	tc.Defs["User"] = ast.TypeDefNode{
		Ident: "User",
		Expr:  ast.TypeDefShapeExpr{Shape: userShape},
	}
	literal := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
			"extra": {
				Type: &ast.TypeNode{Ident: ast.TypeString},
			},
		},
	}
	tc.Functions["g"] = typechecker.FunctionSignature{
		Ident: ast.Ident{ID: "g"},
		Parameters: []typechecker.ParameterSignature{
			{Ident: ast.Ident{ID: "x"}, Type: ast.TypeNode{Ident: "User"}},
		},
	}
	got := tr.getExpectedTypeForShape(&literal, &ShapeContext{
		FunctionName:   "g",
		ParameterIndex: 0,
		ReturnIndex:    -1,
	})
	if got == nil || got.Ident != "User" {
		t.Fatalf("want non-assertion param User via ValidateShapeFields, got %+v", got)
	}
}

func TestGetExpectedTypeForShape_returnTypeSupersetLiteral(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	userShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	tc.Defs["User"] = ast.TypeDefNode{
		Ident: "User",
		Expr:  ast.TypeDefShapeExpr{Shape: userShape},
	}
	literal := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
			"extra": {
				Type: &ast.TypeNode{Ident: ast.TypeString},
			},
		},
	}
	tc.Functions["h"] = typechecker.FunctionSignature{
		Ident:       ast.Ident{ID: "h"},
		ReturnTypes: []ast.TypeNode{{Ident: "User"}},
	}
	got := tr.getExpectedTypeForShape(&literal, &ShapeContext{
		FunctionName:   "h",
		ParameterIndex: -1,
		ReturnIndex:    0,
	})
	if got == nil || got.Ident != "User" {
		t.Fatalf("want return type User via ValidateShapeFields, got %+v", got)
	}
}

func TestGetExpectedTypeForShape_assertionParameterInferErrorSkipsToNil(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	userShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	tc.Defs["User"] = ast.TypeDefNode{
		Ident: "User",
		Expr:  ast.TypeDefShapeExpr{Shape: userShape},
	}
	literal := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
			"extra": {
				Type: &ast.TypeNode{Ident: ast.TypeString},
			},
		},
	}
	missing := ast.TypeIdent("NoSuchType")
	paramType := ast.TypeNode{
		Ident: ast.TypeAssertion,
		Assertion: &ast.AssertionNode{
			BaseType: &missing,
		},
	}
	tc.Functions["bad"] = typechecker.FunctionSignature{
		Ident: ast.Ident{ID: "bad"},
		Parameters: []typechecker.ParameterSignature{
			{Ident: ast.Ident{ID: "x"}, Type: paramType},
		},
	}
	got := tr.getExpectedTypeForShape(&literal, &ShapeContext{
		FunctionName:   "bad",
		ParameterIndex: 0,
		ReturnIndex:    -1,
	})
	if got != nil {
		t.Fatalf("want nil when InferAssertionType fails, got %+v", got)
	}
}

func TestGetExpectedTypeForShape_assertionParameterInferOkValidateFailsReturnsNil(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	userShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	tc.Defs["User"] = ast.TypeDefNode{
		Ident: "User",
		Expr:  ast.TypeDefShapeExpr{Shape: userShape},
	}
	emptyLiteral := ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}

	userIdent := ast.TypeIdent("User")
	paramType := ast.TypeNode{
		Ident: ast.TypeAssertion,
		Assertion: &ast.AssertionNode{
			BaseType: &userIdent,
		},
	}
	tc.Functions["empty"] = typechecker.FunctionSignature{
		Ident: ast.Ident{ID: "empty"},
		Parameters: []typechecker.ParameterSignature{
			{Ident: ast.Ident{ID: "x"}, Type: paramType},
		},
	}
	got := tr.getExpectedTypeForShape(&emptyLiteral, &ShapeContext{
		FunctionName:   "empty",
		ParameterIndex: 0,
		ReturnIndex:    -1,
	})
	if got != nil {
		t.Fatalf("want nil when inferred type ok but ValidateShapeFields fails, got %+v", got)
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

