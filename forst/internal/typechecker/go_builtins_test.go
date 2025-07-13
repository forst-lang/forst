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
	inferredTypes, err := tc.inferExpressionType(ret.Values[0])
	if err != nil {
		t.Fatalf("Failed to infer shape type: %v", err)
	}

	if len(inferredTypes) == 0 {
		t.Fatalf("Expected at least one inferred type")
	}

	inferredType := inferredTypes[0]
	t.Logf("Inferred type from shape literal: %v", inferredType)

	// 6. Test compatibility with the expected return type
	expectedType := ast.TypeNode{Ident: ast.TypeIdent("EchoResponse")}
	compatible := tc.IsTypeCompatible(inferredType, expectedType)

	t.Logf("Inferred type: %v", inferredType)
	t.Logf("Expected type: %v", expectedType)
	t.Logf("Compatible: %v", compatible)

	// This should be false, because the shape literal with Value constraints is more specific
	// than the named type EchoResponse
	if compatible {
		t.Errorf("Expected inferred type to NOT be compatible with EchoResponse (shape literal is more specific)")
	}
}
