package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/hasher"

	"github.com/sirupsen/logrus"
)

func TestInferShapeType_ShouldMatchNamedType(t *testing.T) {
	// Create a typechecker with logger
	tc := New(logrus.New(), false)
	tc.log.SetLevel(4) // Debug level

	// 1. Define EchoResponse type
	echoResponseDef := ast.MakeTypeDef("EchoResponse", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"echo":      ast.MakeTypeField(ast.TypeString),
		"timestamp": ast.MakeTypeField(ast.TypeInt),
	}))

	// 2. Define EchoRequest type
	echoRequestDef := ast.MakeTypeDef("EchoRequest", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"message": ast.MakeTypeField(ast.TypeString),
	}))

	// 3. Create a shape literal that should match EchoResponse
	shapeLiteral := ast.MakeShape(map[string]ast.ShapeFieldNode{
		"echo": {
			Assertion: &ast.AssertionNode{
				Constraints: []ast.ConstraintNode{
					{
						Name: "Value",
						Args: []ast.ConstraintArgumentNode{
							{
								Value: func() *ast.ValueNode {
									v := ast.ValueNode(ast.VariableNode{
										Ident: ast.Ident{ID: ast.Identifier("input.message")},
									})
									return &v
								}(),
							},
						},
					},
				},
			},
		},
		"timestamp": {
			Assertion: &ast.AssertionNode{
				Constraints: []ast.ConstraintNode{
					{
						Name: "Value",
						Args: []ast.ConstraintArgumentNode{
							{
								Value: func() *ast.ValueNode {
									v := ast.ValueNode(ast.IntLiteralNode{Value: 1234567890})
									return &v
								}(),
							},
						},
					},
				},
			},
		},
	})

	// Register the type definitions
	err := tc.CheckTypes([]ast.Node{echoRequestDef, echoResponseDef})
	if err != nil {
		t.Fatalf("Failed to register type definitions: %v", err)
	}

	// Infer the shape type
	expectedType := ast.TypeNode{Ident: "EchoResponse"}
	inferredType, err := tc.inferShapeType(shapeLiteral, &expectedType)
	if err != nil {
		t.Fatalf("Failed to infer shape type: %v", err)
	}

	// The inferred type should be EchoResponse, not a hash-based type
	if inferredType.Ident != expectedType.Ident {
		t.Errorf("Expected inferred type to be %s, but got %s", expectedType.Ident, inferredType.Ident)
	}

	// Also test that the types are compatible
	isCompatible := tc.IsTypeCompatible(inferredType, expectedType)
	if !isCompatible {
		t.Errorf("Expected types to be compatible, but they are not")
	}
}

func TestInferShapeType_FunctionReturnTypeMismatch(t *testing.T) {
	// Create a typechecker with logger
	tc := New(logrus.New(), false)
	tc.log.SetLevel(4) // Debug level

	// 1. Define EchoResponse type
	echoResponseDef := ast.MakeTypeDef("EchoResponse", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"echo":      ast.MakeTypeField(ast.TypeString),
		"timestamp": ast.MakeTypeField(ast.TypeInt),
	}))

	// 2. Define EchoRequest type
	echoRequestDef := ast.MakeTypeDef("EchoRequest", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"message": ast.MakeTypeField(ast.TypeString),
	}))

	// 3. Create a function that returns a shape literal
	fn := ast.MakeFunction("Echo", []ast.ParamNode{
		ast.MakeSimpleParam("input", ast.MakeTypeNode("EchoRequest")),
	}, []ast.Node{
		ast.ReturnNode{
			Values: []ast.ExpressionNode{
				ast.MakeShape(map[string]ast.ShapeFieldNode{
					"echo": {
						Assertion: &ast.AssertionNode{
							Constraints: []ast.ConstraintNode{
								{
									Name: "Value",
									Args: []ast.ConstraintArgumentNode{
										{
											Value: func() *ast.ValueNode {
												v := ast.ValueNode(ast.VariableNode{
													Ident: ast.Ident{ID: ast.Identifier("input.message")},
												})
												return &v
											}(),
										},
									},
								},
							},
						},
					},
					"timestamp": {
						Assertion: &ast.AssertionNode{
							Constraints: []ast.ConstraintNode{
								{
									Name: "Value",
									Args: []ast.ConstraintArgumentNode{
										{
											Value: func() *ast.ValueNode {
												v := ast.ValueNode(ast.IntLiteralNode{Value: 1234567890})
												return &v
											}(),
										},
									},
								},
							},
						},
					},
				}),
			},
		},
	})

	// Set the return type
	fn.ReturnTypes = []ast.TypeNode{{Ident: ast.TypeIdent("EchoResponse")}}

	// Register the type definitions and function
	err := tc.CheckTypes([]ast.Node{echoRequestDef, echoResponseDef, fn})
	if err != nil {
		t.Fatalf("Failed to register types and function: %v", err)
	}

	// The function should be valid - no type mismatch error
	// If there's a type mismatch, CheckTypes will return an error
	t.Logf("Function registered successfully without type mismatch")
}

func TestInferShapeType_ValueVariableAssertion_ShouldResolveToUnderlyingType(t *testing.T) {
	// Create a typechecker with logger
	tc := New(logrus.New(), false)
	tc.log.SetLevel(4) // Debug level

	// 1. Define EchoRequest type with a message field
	echoRequestDef := ast.MakeTypeDef("EchoRequest", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"message": ast.MakeTypeField(ast.TypeString),
	}))

	// 2. Define EchoResponse type
	echoResponseDef := ast.MakeTypeDef("EchoResponse", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"echo":      ast.MakeTypeField(ast.TypeString),
		"timestamp": ast.MakeTypeField(ast.TypeInt),
	}))

	// 3. Create a shape literal with Value(Variable(input.message)) assertion
	shapeLiteral := ast.MakeShape(map[string]ast.ShapeFieldNode{
		"echo": {
			Assertion: &ast.AssertionNode{
				Constraints: []ast.ConstraintNode{
					{
						Name: "Value",
						Args: []ast.ConstraintArgumentNode{
							{
								Value: func() *ast.ValueNode {
									v := ast.ValueNode(ast.VariableNode{
										Ident: ast.Ident{ID: ast.Identifier("input.message")},
									})
									return &v
								}(),
							},
						},
					},
				},
			},
		},
		"timestamp": {
			Assertion: &ast.AssertionNode{
				Constraints: []ast.ConstraintNode{
					{
						Name: "Value",
						Args: []ast.ConstraintArgumentNode{
							{
								Value: func() *ast.ValueNode {
									v := ast.ValueNode(ast.IntLiteralNode{Value: 1234567890})
									return &v
								}(),
							},
						},
					},
				},
			},
		},
	})

	// Register the type definitions
	err := tc.CheckTypes([]ast.Node{echoRequestDef, echoResponseDef})
	if err != nil {
		t.Fatalf("Failed to register type definitions: %v", err)
	}

	// Set up a scope with the input variable
	scope := &Scope{
		Parent: nil,
		Symbols: map[ast.Identifier]Symbol{
			"input": {
				Types: []ast.TypeNode{{Ident: "EchoRequest"}},
				Kind:  SymbolVariable,
			},
		},
	}
	tc.scopeStack = &ScopeStack{
		scopes:  make(map[NodeHash]*Scope),
		current: scope,
		Hasher:  hasher.New(),
		log:     logrus.New(),
	}

	// Infer the shape type
	inferredType, err := tc.inferShapeType(shapeLiteral, nil)
	if err != nil {
		t.Fatalf("Failed to infer shape type: %v", err)
	}

	t.Logf("Inferred type: %s", inferredType.Ident)

	// The inferred type should be EchoResponse, not a hash-based type
	if inferredType.Ident != "EchoResponse" {
		t.Errorf("Expected EchoResponse, got: %s", inferredType.Ident)
	}

	// Verify that the shape is structurally compatible with EchoResponse
	expectedType := ast.TypeNode{Ident: "EchoResponse"}
	compatible := tc.IsTypeCompatible(inferredType, expectedType)
	t.Logf("Type compatibility: %v", compatible)
	if !compatible {
		t.Errorf("Inferred type %s should be compatible with EchoResponse", inferredType.Ident)
	}
}

func TestInferShapeType_EchoExample_ShouldReturnEchoResponse(t *testing.T) {
	// Create a typechecker with logger
	logger := logrus.New()
	if testing.Verbose() {
		logger.SetLevel(logrus.DebugLevel)
	}
	tc := New(logger, false)

	// 1. Define EchoRequest type with a message field
	echoRequestDef := ast.MakeTypeDef("EchoRequest", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"message": ast.MakeTypeField(ast.TypeString),
	}))

	// 2. Define EchoResponse type
	echoResponseDef := ast.MakeTypeDef("EchoResponse", ast.MakeShape(map[string]ast.ShapeFieldNode{
		"echo":      ast.MakeTypeField(ast.TypeString),
		"timestamp": ast.MakeTypeField(ast.TypeInt),
	}))

	// 3. Create a function that returns EchoResponse
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

	// Register the type definitions and function
	err := tc.CheckTypes([]ast.Node{echoRequestDef, echoResponseDef, fn})
	if err != nil {
		t.Fatalf("CheckTypes failed: %v", err)
	}

	// Instead of inferring the return value directly, wrap it in a function and pass it to CheckTypes
	// This simulates the real pipeline and ensures correct scoping

	nodes := []ast.Node{
		echoRequestDef,
		echoResponseDef,
		fn,
	}

	// Run type checking on all nodes (including the function)
	err = tc.CheckTypes(nodes)
	if err != nil {
		t.Fatalf("Type checking failed: %v", err)
	}

	// 4. Check that the function signature is correct
	if len(fn.ReturnTypes) != 1 {
		t.Fatalf("Expected 1 return type, got %d", len(fn.ReturnTypes))
	}

	returnType := fn.ReturnTypes[0]
	if returnType.Ident != "EchoResponse" {
		t.Errorf("Expected EchoResponse, got: %s", returnType.Ident)
	}

	t.Logf("Function return type: %s", returnType.Ident)

}

func TestInferShapeType_ValueNil_UsesExpectedPointerType(t *testing.T) {
	tc := New(logrus.New(), false)

	// Define a shape type with a pointer field
	userType := ast.TypeDefNode{
		Ident: "User",
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"sessionId": {Type: &ast.TypeNode{Ident: "Pointer(String)"}},
				},
			},
		},
	}

	// Create nil literal node
	nilNode := ast.NilLiteralNode{}
	var valueNode ast.ValueNode = nilNode

	// Shape literal with Value(nil) for sessionId
	shapeLiteral := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"sessionId": {
				Assertion: &ast.AssertionNode{
					Constraints: []ast.ConstraintNode{
						{
							Name: "Value",
							Args: []ast.ConstraintArgumentNode{
								{Value: &valueNode},
							},
						},
					},
				},
			},
		},
	}

	nodes := []ast.Node{
		userType,
	}

	// Run type checking on the type definition
	err := tc.CheckTypes(nodes)
	if err != nil {
		t.Fatalf("Type checking failed: %v", err)
	}

	// Now infer the type of the shape literal
	inferredType, err := tc.inferShapeType(shapeLiteral, &ast.TypeNode{Ident: "User"})
	if err != nil {
		t.Fatalf("Failed to infer shape type: %v", err)
	}

	// Check that the inferred type is the named type "User"
	if inferredType.Ident != "User" {
		t.Errorf("Expected User type, got: %s", inferredType.Ident)
	}

	// Check that the sessionId field is a pointer type
	// We need to look up the type definition to check the field type
	if def, ok := tc.Defs["User"]; ok {
		if typeDef, ok := def.(ast.TypeDefNode); ok {
			if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
				if sessionIDField, ok := shapeExpr.Shape.Fields["sessionId"]; ok {
					if sessionIDField.Type != nil {
						if !strings.Contains(string(sessionIDField.Type.Ident), "Pointer") {
							t.Errorf("Expected sessionId field to be a pointer type, got: %s", sessionIDField.Type.Ident)
						}
					} else {
						t.Errorf("sessionId field has no type")
					}
				} else {
					t.Errorf("sessionId field not found in User type definition")
				}
			} else {
				t.Errorf("User type definition is not a shape expression")
			}
		} else {
			t.Errorf("User type definition is not a TypeDefNode")
		}
	} else {
		t.Errorf("User type definition not found")
	}

	t.Logf("Inferred type: %s", inferredType.Ident)
}

// Value(nil) with no contextual type still produces a one-field shape type (hash struct), not a bare Pointer.
func TestInferShapeType_ValueNil_SingleFieldShapeIsNotUnwrappedToPointer(t *testing.T) {
	tc := New(logrus.New(), false)
	if testing.Verbose() {
		tc.log.SetLevel(logrus.DebugLevel)
	}

	// Create nil literal node
	nilNode := ast.NilLiteralNode{}
	var valueNode ast.ValueNode = nilNode

	// Shape literal with Value(nil) for sessionId
	shapeLiteral := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"sessionId": {
				Assertion: &ast.AssertionNode{
					Constraints: []ast.ConstraintNode{
						{
							Name: "Value",
							Args: []ast.ConstraintArgumentNode{
								{Value: &valueNode},
							},
						},
					},
				},
			},
		},
	}

	t.Logf("Shape literal: %+v", shapeLiteral)

	// Infer the shape type
	inferredType, err := tc.inferShapeType(shapeLiteral, nil)
	if err != nil {
		t.Fatalf("Failed to infer shape type: %v", err)
	}

	t.Logf("Inferred type: %s", inferredType.Ident)
	t.Logf("Inferred type details: %+v", inferredType)

	// Single-field `{ sessionId: nil }` is a shape type (hash-named struct), not collapsed to bare Pointer.
	if inferredType.Ident == ast.TypePointer {
		t.Fatalf("expected a struct shape type, not bare %s", ast.TypePointer)
	}
	def, ok := tc.Defs[inferredType.Ident]
	if !ok {
		t.Fatalf("inferred type %q not registered in Defs", inferredType.Ident)
	}
	typeDef, ok := def.(ast.TypeDefNode)
	if !ok {
		t.Fatalf("Defs[%s] is not TypeDefNode", inferredType.Ident)
	}
	shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		t.Fatalf("Defs[%s] is not a shape type", inferredType.Ident)
	}
	sf, ok := shapeExpr.Shape.Fields["sessionId"]
	if !ok {
		t.Fatalf("shape missing sessionId field")
	}
	if sf.Type == nil || sf.Type.Ident != ast.TypePointer || len(sf.Type.TypeParams) != 1 || sf.Type.TypeParams[0].Ident != ast.TypeString {
		t.Fatalf("sessionId field: want Pointer(String), got %+v", sf.Type)
	}
}

// When a struct literal matches both an ordinary shape typedef and a nominal error typedef, bind to the shape.
func TestInferShapeType_prefersShapeTypedefOverNominalErrorWhenBothMatch(t *testing.T) {
	tc := New(logrus.New(), false)
	nodes := []ast.Node{
		ast.TypeDefNode{
			Ident: "ErrDup",
			Expr: ast.TypeDefErrorExpr{
				Payload: ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"code": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
					},
				},
			},
		},
		ast.TypeDefNode{
			Ident: "Row",
			Expr: ast.TypeDefShapeExpr{
				Shape: ast.ShapeNode{
					Fields: map[string]ast.ShapeFieldNode{
						"code": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
					},
				},
			},
		},
	}
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	lit := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"code": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
		},
	}
	out, err := tc.inferShapeType(lit, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.Ident != "Row" {
		t.Fatalf("want named shape typedef Row when ErrDup also matches structurally, got %q", out.Ident)
	}
}
