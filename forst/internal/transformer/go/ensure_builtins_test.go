package transformergo

import (
	goast "go/ast"
	"go/token"
	"testing"

	"forst/internal/ast"
)

func TestTransformBuiltinConstraint_stringContains_emitsStringsContains(t *testing.T) {
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
				{Value: new(ast.ValueNode(ast.StringLiteralNode{Value: "x"}))},
			},
		},
	)
	if err != nil {
		t.Fatalf("TransformBuiltinConstraint: %v", err)
	}
	call, ok := expr.(*goast.CallExpr)
	if !ok {
		t.Fatalf("expected call expr, got %T", expr)
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

func TestTransformBuiltinConstraint_hasPrefixSuffix_andNotEmpty(t *testing.T) {
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)
	at := NewAssertionTransformer(tr)

	for _, name := range []BuiltinConstraint{HasPrefixConstraint, HasSuffixConstraint} {
		expr, err := at.TransformBuiltinConstraint(
			ast.TypeString,
			ast.VariableNode{Ident: ast.Ident{ID: "name"}},
			ast.ConstraintNode{
				Name: string(name),
				Args: []ast.ConstraintArgumentNode{
					{Value: new(ast.ValueNode(ast.StringLiteralNode{Value: "pre"}))},
				},
			},
		)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		call, ok := expr.(*goast.CallExpr)
		if !ok {
			t.Fatalf("%s: expected call, got %T", name, expr)
		}
		sel, ok := call.Fun.(*goast.SelectorExpr)
		if !ok || sel.Sel.Name != string(name) {
			t.Fatalf("%s: got %#v", name, call.Fun)
		}
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
	if !ok || bin.Op != token.NEQ {
		t.Fatalf("expected len(...) != 0 binary expr, got %#v", notEmptyExpr)
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
		{name: "int min", typeIdent: ast.TypeInt, constraint: MinConstraint, arg: ast.IntLiteralNode{Value: 5}, expectedOp: token.GEQ, variableName: "i"},
		{name: "int max", typeIdent: ast.TypeInt, constraint: MaxConstraint, arg: ast.IntLiteralNode{Value: 7}, expectedOp: token.LEQ, variableName: "i"},
		{name: "int lessThan", typeIdent: ast.TypeInt, constraint: LessThanConstraint, arg: ast.IntLiteralNode{Value: 9}, expectedOp: token.LSS, variableName: "i"},
		{name: "int greaterThan", typeIdent: ast.TypeInt, constraint: GreaterThanConstraint, arg: ast.IntLiteralNode{Value: 11}, expectedOp: token.GTR, variableName: "i"},
		{name: "float min", typeIdent: ast.TypeFloat, constraint: MinConstraint, arg: ast.FloatLiteralNode{Value: 1.5}, expectedOp: token.GEQ, variableName: "f"},
		{name: "float max", typeIdent: ast.TypeFloat, constraint: MaxConstraint, arg: ast.FloatLiteralNode{Value: 2.5}, expectedOp: token.LEQ, variableName: "f"},
		{name: "float lessThan", typeIdent: ast.TypeFloat, constraint: LessThanConstraint, arg: ast.FloatLiteralNode{Value: 3.5}, expectedOp: token.LSS, variableName: "f"},
		{name: "float greaterThan", typeIdent: ast.TypeFloat, constraint: GreaterThanConstraint, arg: ast.FloatLiteralNode{Value: 3.5}, expectedOp: token.GTR, variableName: "f"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			expr, err := at.TransformBuiltinConstraint(
				testCase.typeIdent,
				ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(testCase.variableName)}},
				ast.ConstraintNode{
					Name: string(testCase.constraint),
					Args: []ast.ConstraintArgumentNode{
						{Value: new(testCase.arg)},
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

func TestTransformBuiltinConstraint_stringMinMaxBytesAndBool(t *testing.T) {
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)
	at := NewAssertionTransformer(tr)

	minExpr, err := at.TransformBuiltinConstraint(
		ast.TypeString,
		ast.VariableNode{Ident: ast.Ident{ID: "name"}},
		ast.ConstraintNode{
			Name: string(MinConstraint),
			Args: []ast.ConstraintArgumentNode{
				{Value: new(ast.ValueNode(ast.IntLiteralNode{Value: 3}))},
			},
		},
	)
	if err != nil {
		t.Fatalf("string Min: %v", err)
	}
	bin, ok := minExpr.(*goast.BinaryExpr)
	if !ok || bin.Op != token.GEQ {
		t.Fatalf("string Min: got %#v", minExpr)
	}
	call, ok := bin.X.(*goast.CallExpr)
	if !ok {
		t.Fatalf("string Min: expected utf8.RuneCountInString, got %#v", bin.X)
	}
	if sel, ok := call.Fun.(*goast.SelectorExpr); !ok || sel.Sel.Name != "RuneCountInString" {
		t.Fatalf("string Min: expected RuneCountInString, got %#v", call.Fun)
	}

	maxBytesExpr, err := at.TransformBuiltinConstraint(
		ast.TypeString,
		ast.VariableNode{Ident: ast.Ident{ID: "name"}},
		ast.ConstraintNode{
			Name: string(MaxBytesConstraint),
			Args: []ast.ConstraintArgumentNode{
				{Value: new(ast.ValueNode(ast.IntLiteralNode{Value: 10}))},
			},
		},
	)
	if err != nil {
		t.Fatalf("string MaxBytes: %v", err)
	}
	maxBytesBin, ok := maxBytesExpr.(*goast.BinaryExpr)
	if !ok || maxBytesBin.Op != token.LEQ {
		t.Fatalf("string MaxBytes: got %#v", maxBytesExpr)
	}
	if call, ok := maxBytesBin.X.(*goast.CallExpr); !ok {
		t.Fatalf("MaxBytes: expected len call, got %#v", maxBytesBin.X)
	} else if id, ok := call.Fun.(*goast.Ident); !ok || id.Name != "len" {
		t.Fatalf("MaxBytes: expected len, got %#v", call.Fun)
	}

	trueExpr, err := at.TransformBuiltinConstraint(
		ast.TypeBool,
		ast.VariableNode{Ident: ast.Ident{ID: "ok"}},
		ast.ConstraintNode{Name: string(TrueConstraint)},
	)
	if err != nil {
		t.Fatalf("bool True: %v", err)
	}
	if _, ok := trueExpr.(*goast.Ident); !ok {
		if _, isUnary := trueExpr.(*goast.UnaryExpr); isUnary {
			t.Fatalf("bool True: expected variable, got negation")
		}
	}

	falseExpr, err := at.TransformBuiltinConstraint(
		ast.TypeBool,
		ast.VariableNode{Ident: ast.Ident{ID: "ok"}},
		ast.ConstraintNode{Name: string(FalseConstraint)},
	)
	if err != nil {
		t.Fatalf("bool False: %v", err)
	}
	if _, ok := falseExpr.(*goast.UnaryExpr); !ok {
		t.Fatalf("bool False: expected negation, got %T", falseExpr)
	}
}

func TestTransformBuiltinConstraint_arrayMapBytesMinMax(t *testing.T) {
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)
	at := NewAssertionTransformer(tr)

	for _, typeIdent := range []ast.TypeIdent{ast.TypeArray, ast.TypeMap, ast.TypeBytes} {
		minExpr, err := at.TransformBuiltinConstraint(
			typeIdent,
			ast.VariableNode{Ident: ast.Ident{ID: "xs"}},
			ast.ConstraintNode{
				Name: string(MinConstraint),
				Args: []ast.ConstraintArgumentNode{
					{Value: new(ast.ValueNode(ast.IntLiteralNode{Value: 2}))},
				},
			},
		)
		if err != nil {
			t.Fatalf("%s Min: %v", typeIdent, err)
		}
		if bin, ok := minExpr.(*goast.BinaryExpr); !ok || bin.Op != token.GEQ {
			t.Fatalf("%s Min: got %#v", typeIdent, minExpr)
		}

		maxExpr, err := at.TransformBuiltinConstraint(
			typeIdent,
			ast.VariableNode{Ident: ast.Ident{ID: "xs"}},
			ast.ConstraintNode{
				Name: string(MaxConstraint),
				Args: []ast.ConstraintArgumentNode{
					{Value: new(ast.ValueNode(ast.IntLiteralNode{Value: 5}))},
				},
			},
		)
		if err != nil {
			t.Fatalf("%s Max: %v", typeIdent, err)
		}
		if bin, ok := maxExpr.(*goast.BinaryExpr); !ok || bin.Op != token.LEQ {
			t.Fatalf("%s Max: got %#v", typeIdent, maxExpr)
		}
	}
}

func TestTransformBuiltinConstraint_finite(t *testing.T) {
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)
	at := NewAssertionTransformer(tr)

	expr, err := at.TransformBuiltinConstraint(
		ast.TypeFloat,
		ast.VariableNode{Ident: ast.Ident{ID: "f"}},
		ast.ConstraintNode{Name: string(FiniteConstraint)},
	)
	if err != nil {
		t.Fatalf("Finite: %v", err)
	}
	bin, ok := expr.(*goast.BinaryExpr)
	if !ok || bin.Op != token.LAND {
		t.Fatalf("expected !IsInf && !IsNaN, got %#v", expr)
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
		{name: "pointer nil", typeIdent: ast.TypePointer, constraint: NilConstraint, expectedOp: token.EQL, variableName: "ptr"},
		{name: "pointer present", typeIdent: ast.TypePointer, constraint: PresentConstraint, expectedOp: token.NEQ, variableName: "ptr"},
		{name: "error nil", typeIdent: ast.TypeError, constraint: NilConstraint, expectedOp: token.EQL, variableName: "err"},
		{name: "error present", typeIdent: ast.TypeError, constraint: PresentConstraint, expectedOp: token.NEQ, variableName: "err"},
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

func TestTransformBuiltinConstraint_rejectsWrongCarrier(t *testing.T) {
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)
	at := NewAssertionTransformer(tr)

	if _, err := at.TransformBuiltinConstraint(
		ast.TypeInt,
		ast.VariableNode{Ident: ast.Ident{ID: "n"}},
		ast.ConstraintNode{Name: string(FiniteConstraint)},
	); err == nil {
		t.Fatal("expected Finite on Int to fail")
	}
	if _, err := at.TransformBuiltinConstraint(
		ast.TypeInt,
		ast.VariableNode{Ident: ast.Ident{ID: "n"}},
		ast.ConstraintNode{
			Name: string(HasSuffixConstraint),
			Args: []ast.ConstraintArgumentNode{
				{Value: new(ast.ValueNode(ast.StringLiteralNode{Value: ".md"}))},
			},
		},
	); err == nil {
		t.Fatal("expected HasSuffix on Int to fail")
	}
	if _, err := at.TransformBuiltinConstraint(
		ast.TypeInt,
		ast.VariableNode{Ident: ast.Ident{ID: "n"}},
		ast.ConstraintNode{
			Name: string(MinBytesConstraint),
			Args: []ast.ConstraintArgumentNode{
				{Value: new(ast.ValueNode(ast.IntLiteralNode{Value: 1}))},
			},
		},
	); err == nil {
		t.Fatal("expected MinBytes on Int to fail")
	}
}
