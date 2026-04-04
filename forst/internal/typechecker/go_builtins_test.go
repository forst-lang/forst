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

func TestCheckBuiltinFunctionCall_stringRejectsNonInt(t *testing.T) {
	tc := New(logrus.New(), false)
	fn := BuiltinFunctions["string"]
	_, err := tc.checkBuiltinFunctionCall(fn, []ast.ExpressionNode{ast.StringLiteralNode{Value: "x"}}, nil, ast.SourceSpan{})
	if err == nil {
		t.Fatal("expected error for string() of string")
	}
}
