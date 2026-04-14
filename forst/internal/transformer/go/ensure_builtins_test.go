package transformergo

import (
	goast "go/ast"
	"go/token"
	"testing"

	"forst/internal/ast"
)

func valueNode(v ast.ValueNode) *ast.ValueNode {
	return &v
}

func TestTransformBuiltinConstraint_stringContains_emitsNegatedStringsContains(t *testing.T) {
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)
	at := NewAssertionTransformer(tr)

	expr, err := at.TransformBuiltinConstraint(
		ast.TypeString,
		ast.VariableNode{Ident: ast.Ident{ID: "name"}},
		ast.ConstraintNode{
			Name: string(ContainsConstraint),
			Args: []ast.ConstraintArgumentNode{
				{Value: valueNode(ast.StringLiteralNode{Value: "x"})},
			},
		},
	)
	if err != nil {
		t.Fatalf("TransformBuiltinConstraint: %v", err)
	}
	u, ok := expr.(*goast.UnaryExpr)
	if !ok {
		t.Fatalf("expected negated unary expr, got %T", expr)
	}
	call, ok := u.X.(*goast.CallExpr)
	if !ok {
		t.Fatalf("expected call inside negation, got %T", u.X)
	}
	sel, ok := call.Fun.(*goast.SelectorExpr)
	if !ok || sel.Sel.Name != "Contains" {
		t.Fatalf("expected strings.Contains call, got %#v", call.Fun)
	}
}

func TestTransformBuiltinConstraint_unknownTypeOrConstraint(t *testing.T) {
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)
	at := NewAssertionTransformer(tr)

	if _, err := at.TransformBuiltinConstraint(
		ast.TypeIdent("TYPE_UNKNOWN"),
		ast.VariableNode{Ident: ast.Ident{ID: "x"}},
		ast.ConstraintNode{Name: string(MinConstraint)},
	); err == nil {
		t.Fatal("expected unknown typeIdent error")
	}

	if _, err := at.TransformBuiltinConstraint(
		ast.TypeString,
		ast.VariableNode{Ident: ast.Ident{ID: "x"}},
		ast.ConstraintNode{Name: "NoSuchConstraint"},
	); err == nil {
		t.Fatal("expected unknown constraint error")
	}
}

func TestTransformBuiltinConstraint_hasPrefix_andNotEmpty(t *testing.T) {
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)
	at := NewAssertionTransformer(tr)

	prefixExpr, err := at.TransformBuiltinConstraint(
		ast.TypeString,
		ast.VariableNode{Ident: ast.Ident{ID: "name"}},
		ast.ConstraintNode{
			Name: string(HasPrefixConstraint),
			Args: []ast.ConstraintArgumentNode{
				{Value: valueNode(ast.StringLiteralNode{Value: "pre"})},
			},
		},
	)
	if err != nil {
		t.Fatalf("HasPrefix: %v", err)
	}
	if _, ok := prefixExpr.(*goast.UnaryExpr); !ok {
		t.Fatalf("expected negated expression for HasPrefix, got %T", prefixExpr)
	}

	notEmptyExpr, err := at.TransformBuiltinConstraint(
		ast.TypeArray,
		ast.VariableNode{Ident: ast.Ident{ID: "items"}},
		ast.ConstraintNode{Name: string(NotEmptyConstraint)},
	)
	if err != nil {
		t.Fatalf("NotEmpty: %v", err)
	}
	bin, ok := notEmptyExpr.(*goast.BinaryExpr)
	if !ok || bin.Op != token.EQL {
		t.Fatalf("expected len(...) == 0 binary expr, got %#v", notEmptyExpr)
	}
}

func TestTransformBuiltinConstraint_numericComparators(t *testing.T) {
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)
	at := NewAssertionTransformer(tr)

	testCases := []struct {
		name         string
		typeIdent    ast.TypeIdent
		constraint   BuiltinConstraint
		arg          ast.ValueNode
		expectedOp   token.Token
		variableName string
	}{
		{name: "int min", typeIdent: ast.TypeInt, constraint: MinConstraint, arg: ast.IntLiteralNode{Value: 5}, expectedOp: token.LSS, variableName: "i"},
		{name: "int max", typeIdent: ast.TypeInt, constraint: MaxConstraint, arg: ast.IntLiteralNode{Value: 7}, expectedOp: token.GTR, variableName: "i"},
		{name: "int lessThan", typeIdent: ast.TypeInt, constraint: LessThanConstraint, arg: ast.IntLiteralNode{Value: 9}, expectedOp: token.GEQ, variableName: "i"},
		{name: "int greaterThan", typeIdent: ast.TypeInt, constraint: GreaterThanConstraint, arg: ast.IntLiteralNode{Value: 11}, expectedOp: token.LEQ, variableName: "i"},
		{name: "float min", typeIdent: ast.TypeFloat, constraint: MinConstraint, arg: ast.FloatLiteralNode{Value: 1.5}, expectedOp: token.LSS, variableName: "f"},
		{name: "float max", typeIdent: ast.TypeFloat, constraint: MaxConstraint, arg: ast.FloatLiteralNode{Value: 2.5}, expectedOp: token.GTR, variableName: "f"},
		{name: "float greaterThan", typeIdent: ast.TypeFloat, constraint: GreaterThanConstraint, arg: ast.FloatLiteralNode{Value: 3.5}, expectedOp: token.LEQ, variableName: "f"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			expr, err := at.TransformBuiltinConstraint(
				testCase.typeIdent,
				ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(testCase.variableName)}},
				ast.ConstraintNode{
					Name: string(testCase.constraint),
					Args: []ast.ConstraintArgumentNode{
						{Value: valueNode(testCase.arg)},
					},
				},
			)
			if err != nil {
				t.Fatalf("TransformBuiltinConstraint: %v", err)
			}
			bin, ok := expr.(*goast.BinaryExpr)
			if !ok {
				t.Fatalf("expected binary expr, got %T", expr)
			}
			if bin.Op != testCase.expectedOp {
				t.Fatalf("unexpected operator: got %s want %s", bin.Op, testCase.expectedOp)
			}
		})
	}
}

func TestTransformBuiltinConstraint_nilAndPresent(t *testing.T) {
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)
	at := NewAssertionTransformer(tr)

	testCases := []struct {
		name         string
		typeIdent    ast.TypeIdent
		constraint   BuiltinConstraint
		expectedOp   token.Token
		variableName string
	}{
		{name: "pointer nil", typeIdent: ast.TypePointer, constraint: NilConstraint, expectedOp: token.NEQ, variableName: "ptr"},
		{name: "pointer present", typeIdent: ast.TypePointer, constraint: PresentConstraint, expectedOp: token.EQL, variableName: "ptr"},
		{name: "error nil", typeIdent: ast.TypeError, constraint: NilConstraint, expectedOp: token.NEQ, variableName: "err"},
		{name: "error present", typeIdent: ast.TypeError, constraint: PresentConstraint, expectedOp: token.EQL, variableName: "err"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			expr, err := at.TransformBuiltinConstraint(
				testCase.typeIdent,
				ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(testCase.variableName)}},
				ast.ConstraintNode{Name: string(testCase.constraint)},
			)
			if err != nil {
				t.Fatalf("TransformBuiltinConstraint: %v", err)
			}
			bin, ok := expr.(*goast.BinaryExpr)
			if !ok {
				t.Fatalf("expected binary expr, got %T", expr)
			}
			if bin.Op != testCase.expectedOp {
				t.Fatalf("unexpected operator: got %s want %s", bin.Op, testCase.expectedOp)
			}
		})
	}
}
