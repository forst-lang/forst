package typechecker

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestReturnTypeInference_ShapeLiteralVsNamedType(t *testing.T) {
	tc := New(logrus.New(), false)

	// 1. EchoRequest type
	echoRequestDef := ast.TypeDefNode{
		Ident: ast.TypeIdent("EchoRequest"),
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"message": {
						Type: &ast.TypeNode{Ident: ast.TypeString},
					},
				},
			},
		},
	}

	// 2. EchoResponse type
	echoResponseDef := ast.TypeDefNode{
		Ident: ast.TypeIdent("EchoResponse"),
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"echo": {
						Type: &ast.TypeNode{Ident: ast.TypeString},
					},
					"timestamp": {
						Type: &ast.TypeNode{Ident: ast.TypeInt},
					},
				},
			},
		},
	}

	// 3. Function node: func Echo(input EchoRequest): EchoResponse { return { echo: input.message, timestamp: 1234567890 } }
	fn := ast.FunctionNode{
		Ident: ast.Ident{ID: "Echo"},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: "input"},
				Type:  ast.TypeNode{Ident: ast.TypeIdent("EchoRequest")},
			},
		},
		ReturnTypes: []ast.TypeNode{{Ident: ast.TypeIdent("EchoResponse")}},
		Body: []ast.Node{
			ast.ReturnNode{
				Values: []ast.ExpressionNode{
					ast.ShapeNode{
						Fields: map[string]ast.ShapeFieldNode{
							"echo": {
								Assertion: &ast.AssertionNode{
									Constraints: []ast.ConstraintNode{{
										Name: "Value",
										Args: []ast.ConstraintArgumentNode{{
											Value: func() *ast.ValueNode {
												v := ast.ValueNode(ast.VariableNode{
													Ident: ast.Ident{ID: "input.message"},
												})
												return &v
											}(),
										}},
									}},
								},
							},
							"timestamp": {
								Assertion: &ast.AssertionNode{
									Constraints: []ast.ConstraintNode{{
										Name: "Value",
										Args: []ast.ConstraintArgumentNode{{
											Value: func() *ast.ValueNode {
												v := ast.ValueNode(ast.IntLiteralNode{Value: 1234567890})
												return &v
											}(),
										}},
									}},
								},
							},
						},
					},
				},
			},
		},
	}

	// 4. Use CheckTypes to register all top-level nodes
	err := tc.CheckTypes([]ast.Node{echoRequestDef, echoResponseDef, fn})
	if err != nil {
		t.Fatalf("CheckTypes failed: %v", err)
	}

	// 5. Infer the type of the return value (the shape literal)
	ret := fn.Body[0].(ast.ReturnNode)
	shapeLiteral := ret.Values[0].(ast.ShapeNode)

	// Pass the expected return type to help with Value constraint inference
	expectedReturnType := ast.TypeNode{Ident: ast.TypeIdent("EchoResponse")}
	inferredType, err := tc.inferShapeType(shapeLiteral, &expectedReturnType)
	if err != nil {
		t.Fatalf("Failed to infer shape type: %v", err)
	}

	inferredTypes := []ast.TypeNode{inferredType}

	if len(inferredTypes) == 0 {
		t.Fatalf("Expected at least one inferred type")
	}

	inferredTypeResult := inferredTypes[0]
	t.Logf("Inferred type from shape literal: %v", inferredTypeResult)

	// 6. Test compatibility with the expected return type
	expectedType := ast.TypeNode{Ident: ast.TypeIdent("EchoResponse")}
	compatible := tc.IsTypeCompatible(inferredTypeResult, expectedType)

	t.Logf("Inferred type: %v", inferredTypeResult)
	t.Logf("Expected type: %v", expectedType)
	t.Logf("Compatible: %v", compatible)

	// This should be true, because the shape literal matches the named type EchoResponse exactly
	if !compatible {
		t.Errorf("Expected inferred type to be compatible with EchoResponse (shape literal matches named type)")
	}
}

func TestCheckBuiltinFunctionCall_stringInt(t *testing.T) {
	tc := New(logrus.New(), false)
	fn, ok := BuiltinFunctions["string"]
	if !ok {
		t.Fatal("missing string builtin")
	}
	types, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{ast.IntLiteralNode{Value: 7}}, nil, ast.SourceSpan{})
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != 1 || types[0].Ident != ast.TypeString {
		t.Fatalf("got %+v", types)
	}
}

func TestIsTypeCompatible_concreteAssignableToTypeObject(t *testing.T) {
	tc := New(logrus.New(), false)
	if !tc.IsTypeCompatible(ast.NewBuiltinType(ast.TypeString), ast.NewBuiltinType(ast.TypeObject)) {
		t.Fatal("String should be assignable where TypeObject is expected (empty-interface-like)")
	}
	if tc.IsTypeCompatible(ast.NewBuiltinType(ast.TypeVoid), ast.NewBuiltinType(ast.TypeObject)) {
		t.Fatal("Void should not assign to TypeObject")
	}
}

func TestBuiltinFunction_lenIsDispatchKind(t *testing.T) {
	if BuiltinFunctions["len"].CheckKind != BuiltinCheckDispatch {
		t.Fatalf("len should use BuiltinCheckDispatch, got %v", BuiltinFunctions["len"].CheckKind)
	}
	if BuiltinFunctions["string"].CheckKind != BuiltinCheckGeneric {
		t.Fatalf("string() should use BuiltinCheckGeneric, got %v", BuiltinFunctions["string"].CheckKind)
	}
}

func TestCheckBuiltinFunctionCall_stringRejectsNonInt(t *testing.T) {
	tc := New(logrus.New(), false)
	fn := BuiltinFunctions["string"]
	_, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{ast.StringLiteralNode{Value: "x"}}, nil, ast.SourceSpan{})
	if err == nil {
		t.Fatal("expected error for string() of string")
	}
}

func TestCheckBuiltinFunctionCall_goPredeclared(t *testing.T) {
	tc := New(logrus.New(), false)
	tc.CurrentScope().RegisterSymbol(ast.Identifier("m"), []ast.TypeNode{
		ast.NewMapType(ast.NewBuiltinType(ast.TypeString), ast.NewBuiltinType(ast.TypeInt)),
	}, SymbolVariable)

	t.Run("len string ok", func(t *testing.T) {
		fn := BuiltinFunctions["len"]
		types, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{ast.StringLiteralNode{Value: "ab"}}, nil, ast.SourceSpan{})
		if err != nil {
			t.Fatal(err)
		}
		if len(types) != 1 || types[0].Ident != ast.TypeInt {
			t.Fatalf("got %+v", types)
		}
	})

	t.Run("len string variable", func(t *testing.T) {
		tc2 := New(logrus.New(), false)
		tc2.CurrentScope().RegisterSymbol(ast.Identifier("s"), []ast.TypeNode{ast.NewBuiltinType(ast.TypeString)}, SymbolVariable)
		fn := BuiltinFunctions["len"]
		types, err := tc2.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "s"}}}, nil, ast.SourceSpan{})
		if err != nil {
			t.Fatal(err)
		}
		if len(types) != 1 || types[0].Ident != ast.TypeInt {
			t.Fatalf("got %+v", types)
		}
	})

	t.Run("len int fails", func(t *testing.T) {
		fn := BuiltinFunctions["len"]
		_, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{ast.IntLiteralNode{Value: 1}}, nil, ast.SourceSpan{})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("append slice", func(t *testing.T) {
		fn := BuiltinFunctions["append"]
		types, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{
			ast.ArrayLiteralNode{Value: []ast.LiteralNode{ast.IntLiteralNode{Value: 1}}},
			ast.IntLiteralNode{Value: 2},
		}, nil, ast.SourceSpan{})
		if err != nil {
			t.Fatal(err)
		}
		if len(types) != 1 || types[0].Ident != ast.TypeArray {
			t.Fatalf("got %+v", types)
		}
	})

	t.Run("delete map", func(t *testing.T) {
		fn := BuiltinFunctions["delete"]
		_, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{
			ast.VariableNode{Ident: ast.Ident{ID: "m"}},
			ast.StringLiteralNode{Value: "k"},
		}, nil, ast.SourceSpan{})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("min int", func(t *testing.T) {
		fn := BuiltinFunctions["min"]
		types, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{
			ast.IntLiteralNode{Value: 3},
			ast.IntLiteralNode{Value: 1},
		}, nil, ast.SourceSpan{})
		if err != nil {
			t.Fatal(err)
		}
		if len(types) != 1 || types[0].Ident != ast.TypeInt {
			t.Fatalf("got %+v", types)
		}
	})

	t.Run("min mixed types fails", func(t *testing.T) {
		fn := BuiltinFunctions["min"]
		_, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{
			ast.IntLiteralNode{Value: 1},
			ast.FloatLiteralNode{Value: 2},
		}, nil, ast.SourceSpan{})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("complex", func(t *testing.T) {
		fn := BuiltinFunctions["complex"]
		types, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{
			ast.FloatLiteralNode{Value: 1},
			ast.FloatLiteralNode{Value: 0},
		}, nil, ast.SourceSpan{})
		if err != nil {
			t.Fatal(err)
		}
		if len(types) != 1 || types[0].Ident != ast.TypeObject {
			t.Fatalf("got %+v", types)
		}
	})

	t.Run("real of complex", func(t *testing.T) {
		complexCall := ast.FunctionCallNode{
			Function: ast.Ident{ID: "complex"},
			Arguments: []ast.ExpressionNode{
				ast.FloatLiteralNode{Value: 3},
				ast.FloatLiteralNode{Value: 4},
			},
		}
		fn := BuiltinFunctions["real"]
		types, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{complexCall}, nil, ast.SourceSpan{})
		if err != nil {
			t.Fatal(err)
		}
		if len(types) != 1 || types[0].Ident != ast.TypeFloat {
			t.Fatalf("got %+v", types)
		}
	})

	t.Run("recover", func(t *testing.T) {
		fn := BuiltinFunctions["recover"]
		types, err := tc.checkBuiltinFunctionCall(fn, nil, nil, ast.SourceSpan{})
		if err != nil {
			t.Fatal(err)
		}
		if len(types) != 1 || types[0].Ident != ast.TypeObject {
			t.Fatalf("got %+v", types)
		}
	})

	t.Run("make unsupported", func(t *testing.T) {
		fn := BuiltinFunctions["make"]
		_, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{ast.StringLiteralNode{Value: "x"}}, nil, ast.SourceSpan{})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("println mixed", func(t *testing.T) {
		fn := BuiltinFunctions["println"]
		_, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{
			ast.StringLiteralNode{Value: "a"},
			ast.IntLiteralNode{Value: 2},
		}, nil, ast.SourceSpan{})
		if err != nil {
			t.Fatal(err)
		}
	})
}
