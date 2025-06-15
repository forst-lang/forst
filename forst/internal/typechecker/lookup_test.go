package typechecker

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestTypeAliasLookup(t *testing.T) {
	tc := New(logrus.New(), false)

	// Create type alias: Password = String
	alias := ast.TypeDefNode{
		Ident: ast.TypeIdent("Password"),
		Expr: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: typeIdentPtr(string(ast.TypeString)),
			},
		},
	}

	// Register the type alias
	tc.Defs[alias.Ident] = alias

	// Test that Password resolves to String
	passwordType := ast.TypeNode{Ident: ast.TypeIdent("Password")}
	resolved := tc.IsTypeCompatible(passwordType, ast.TypeNode{Ident: ast.TypeString})
	if !resolved {
		t.Errorf("expected Password to be compatible with String")
	}
}

func TestTypeGuardLookup(t *testing.T) {
	tc := New(logrus.New(), false)

	// Create type guard: is (password Password) Strong
	guard := ast.TypeGuardNode{
		Ident: ast.Identifier("Strong"),
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: ast.Identifier("password")},
			Type:  ast.TypeNode{Ident: ast.TypeIdent("Password")},
		},
		Body: []ast.Node{
			ast.IfNode{
				Condition: ast.BinaryExpressionNode{
					Left:     ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier("password")}},
					Operator: ast.TokenIs,
					Right: ast.AssertionNode{
						BaseType: typeIdentPtr(string(ast.TypeString)),
						Constraints: []ast.ConstraintNode{
							{
								Name: "Min",
								Args: []ast.ConstraintArgumentNode{
									{
										Value: func() *ast.ValueNode { v := ast.IntLiteralNode{Value: 12}; var n ast.ValueNode = v; return &n }(),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Register the type guard in the global scope
	globalScope := tc.scopeStack.globalScope()
	globalScope.RegisterSymbol(guard.Ident, []ast.TypeNode{{Ident: ast.TypeVoid}}, SymbolTypeGuard)

	// Also register it in Defs for type lookup
	tc.Defs[ast.TypeIdent(guard.Ident)] = guard

	// Test that the type guard can be looked up
	guardType := ast.TypeNode{Ident: ast.TypeIdent(guard.Ident)}
	if _, exists := tc.Defs[guardType.Ident]; !exists {
		t.Errorf("expected type guard %s to be registered", guard.Ident)
	}

	// Test that the type guard is in the global scope with correct kind
	symbol, exists := globalScope.Symbols[guard.Ident]
	if !exists {
		t.Errorf("expected type guard %s to be found in global scope", guard.Ident)
	}
	if symbol.Kind != SymbolTypeGuard {
		t.Errorf("expected type guard to have kind 'type_guard' but got '%v'", symbol.Kind)
	}
	if symbol.Types[0].Ident != ast.TypeVoid {
		t.Errorf("expected type guard to have type 'void' but got '%s'", symbol.Types[0].Ident)
	}
}

func TestVariableTypeLookup(t *testing.T) {
	tc := New(logrus.New(), false)

	// Create type alias: Password = String
	alias := ast.TypeDefNode{
		Ident: ast.TypeIdent("Password"),
		Expr: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: typeIdentPtr(string(ast.TypeString)),
			},
		},
	}

	// Register the type alias
	tc.Defs[alias.Ident] = alias

	// Create a variable of type Password
	variable := ast.VariableNode{
		Ident: ast.Ident{ID: ast.Identifier("password")},
		ExplicitType: ast.TypeNode{
			Ident: ast.TypeIdent("Password"),
		},
	}

	// Create a scope and store the variable
	scope := NewScope(nil, nil, logrus.New())
	scope.RegisterSymbol(variable.Ident.ID, []ast.TypeNode{variable.ExplicitType}, SymbolVariable)

	// Test that the variable's type can be looked up
	lookedUpType, err := tc.LookupVariableType(&variable, scope)
	if err != nil {
		t.Errorf("unexpected error looking up variable type: %v", err)
	}
	if lookedUpType.Ident != variable.ExplicitType.Ident {
		t.Errorf("expected variable type %s, got %s", variable.ExplicitType.Ident, lookedUpType.Ident)
	}
}

func TestEnsureTypeLookup(t *testing.T) {
	tc := New(logrus.New(), false)

	// Create type alias: Password = String
	alias := ast.TypeDefNode{
		Ident: ast.TypeIdent("Password"),
		Expr: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: typeIdentPtr(string(ast.TypeString)),
			},
		},
	}

	// Register the type alias
	tc.Defs[alias.Ident] = alias

	// Create a type guard: is (password Password) Strong
	guard := ast.TypeGuardNode{
		Ident: ast.Identifier("Strong"),
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: ast.Identifier("password")},
			Type:  ast.TypeNode{Ident: ast.TypeIdent("Password")},
		},
		Body: []ast.Node{
			ast.IfNode{
				Condition: ast.BinaryExpressionNode{
					Left:     ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier("password")}},
					Operator: ast.TokenIs,
					Right: ast.AssertionNode{
						BaseType: typeIdentPtr(string(ast.TypeString)),
						Constraints: []ast.ConstraintNode{
							{
								Name: "Min",
								Args: []ast.ConstraintArgumentNode{
									{
										Value: func() *ast.ValueNode { v := ast.IntLiteralNode{Value: 12}; var n ast.ValueNode = v; return &n }(),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Register the type guard
	tc.Defs[ast.TypeIdent(guard.Ident)] = guard

	// Create an ensure statement
	ensure := ast.EnsureNode{
		Variable: ast.VariableNode{
			Ident: ast.Ident{ID: ast.Identifier("password")},
			ExplicitType: ast.TypeNode{
				Ident: ast.TypeIdent("Password"),
			},
		},
		Assertion: ast.AssertionNode{
			BaseType: typeIdentPtr(string(ast.TypeIdent("Strong"))),
		},
	}

	// Test that the ensure statement's type can be looked up
	assertionType, err := tc.LookupAssertionType(&ensure.Assertion)
	if err != nil {
		t.Errorf("unexpected error looking up assertion type: %v", err)
	}
	if assertionType.Ident != ast.TypeIdent(guard.Ident) {
		t.Errorf("expected assertion type %s, got %s", guard.Ident, assertionType.Ident)
	}
}

func TestTypeInference(t *testing.T) {
	tc := New(logrus.New(), false)

	// Create a string literal
	literal := ast.StringLiteralNode{Value: "test"}

	// Infer the type for the literal (registers the type)
	_, err := tc.inferExpressionType(literal)
	if err != nil {
		t.Errorf("unexpected error during type inference: %v", err)
	}

	// Test that the literal's type is inferred as String
	inferredTypes, err := tc.LookupInferredType(literal, true)
	if err != nil {
		t.Errorf("unexpected error inferring type: %v", err)
	}
	if len(inferredTypes) != 1 {
		t.Errorf("expected 1 inferred type, got %d", len(inferredTypes))
	}
	if inferredTypes[0].Ident != ast.TypeString {
		t.Errorf("expected inferred type String, got %s", inferredTypes[0].Ident)
	}
}

func TestGetTypeAliasChain(t *testing.T) {
	tc := New(logrus.New(), false)

	// Create type aliases:
	// Password = String
	// StrongPassword = Password
	// VeryStrongPassword = StrongPassword
	stringType := ast.TypeString
	passwordType := ast.TypeIdent("Password")
	strongPasswordType := ast.TypeIdent("StrongPassword")
	veryStrongPasswordType := ast.TypeIdent("VeryStrongPassword")

	passwordAlias := ast.TypeDefNode{
		Ident: passwordType,
		Expr: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: &stringType,
			},
		},
	}

	strongPasswordAlias := ast.TypeDefNode{
		Ident: strongPasswordType,
		Expr: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: &passwordType,
			},
		},
	}

	veryStrongPasswordAlias := ast.TypeDefNode{
		Ident: veryStrongPasswordType,
		Expr: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: &strongPasswordType,
			},
		},
	}

	// Register the type aliases
	tc.Defs[passwordAlias.Ident] = passwordAlias
	tc.Defs[strongPasswordAlias.Ident] = strongPasswordAlias
	tc.Defs[veryStrongPasswordAlias.Ident] = veryStrongPasswordAlias

	tests := []struct {
		name     string
		start    ast.TypeNode
		expected []ast.TypeNode
	}{
		{
			name:     "direct alias",
			start:    ast.TypeNode{Ident: passwordType},
			expected: []ast.TypeNode{{Ident: passwordType}, {Ident: ast.TypeString}},
		},
		{
			name:     "two level alias",
			start:    ast.TypeNode{Ident: strongPasswordType},
			expected: []ast.TypeNode{{Ident: strongPasswordType}, {Ident: passwordType}, {Ident: ast.TypeString}},
		},
		{
			name:     "three level alias",
			start:    ast.TypeNode{Ident: veryStrongPasswordType},
			expected: []ast.TypeNode{{Ident: veryStrongPasswordType}, {Ident: strongPasswordType}, {Ident: passwordType}, {Ident: ast.TypeString}},
		},
		{
			name:     "non-alias type",
			start:    ast.TypeNode{Ident: ast.TypeString},
			expected: []ast.TypeNode{{Ident: ast.TypeString}},
		},
		{
			name:     "unknown type",
			start:    ast.TypeNode{Ident: ast.TypeIdent("Unknown")},
			expected: []ast.TypeNode{{Ident: ast.TypeIdent("Unknown")}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := tc.GetTypeAliasChain(tt.start)
			if len(chain) != len(tt.expected) {
				t.Errorf("expected chain length %d, got %d", len(tt.expected), len(chain))
				return
			}
			for i, expected := range tt.expected {
				if chain[i].Ident != expected.Ident {
					t.Errorf("at index %d: expected %v, got %v", i, expected.Ident, chain[i].Ident)
				}
			}
		})
	}
}

func TestShapeRefinementLookup(t *testing.T) {
	tc := New(logrus.New(), false)

	// Create base shape type
	baseShapeType := ast.TypeIdent("BaseShape")
	baseShape := ast.TypeDefNode{
		Ident: baseShapeType,
		Expr: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: typeIdentPtr(string(ast.TypeShape)),
			},
		},
	}
	tc.Defs[baseShapeType] = baseShape

	// Create a type guard that refines the shape with a field
	guard := ast.TypeGuardNode{
		Ident: ast.Identifier("HasName"),
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: ast.Identifier("s")},
			Type:  ast.TypeNode{Ident: baseShapeType},
		},
		Body: []ast.Node{
			ast.EnsureNode{
				Variable: ast.VariableNode{
					Ident: ast.Ident{ID: ast.Identifier("s")},
				},
				Assertion: ast.AssertionNode{
					BaseType: typeIdentPtr(string(ast.TypeShape)),
					Constraints: []ast.ConstraintNode{
						{
							Name: "HasField",
							Args: []ast.ConstraintArgumentNode{
								{
									Value: func() *ast.ValueNode { v := ast.StringLiteralNode{Value: "name"}; var n ast.ValueNode = v; return &n }(),
								},
							},
						},
					},
				},
			},
		},
	}

	// Register the type guard in the global scope
	globalScope := tc.scopeStack.globalScope()
	globalScope.RegisterSymbol(guard.Ident, []ast.TypeNode{{Ident: ast.TypeVoid}}, SymbolTypeGuard)

	// Also register it in Defs for type lookup
	tc.Defs[ast.TypeIdent(guard.Ident)] = guard

	// Create a type alias that uses the type guard
	refinedShapeType := ast.TypeIdent("RefinedShape")
	refinedShape := ast.TypeDefNode{
		Ident: refinedShapeType,
		Expr: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: &baseShapeType,
				Constraints: []ast.ConstraintNode{
					{
						Name: string(guard.Ident),
					},
				},
			},
		},
	}
	tc.Defs[refinedShapeType] = refinedShape

	// Register the shape type with the name field
	shapeType := ast.TypeDefNode{
		Ident: refinedShapeType,
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"name": {
						Type: &ast.TypeNode{Ident: ast.TypeString},
					},
				},
			},
		},
	}
	tc.Defs[refinedShapeType] = shapeType

	// Simulate a function scope and declare variable s of type RefinedShape
	scope := NewScope(nil, nil, logrus.New())
	sVar := ast.VariableNode{
		Ident:        ast.Ident{ID: ast.Identifier("s")},
		ExplicitType: ast.TypeNode{Ident: refinedShapeType},
	}
	scope.RegisterSymbol(sVar.Ident.ID, []ast.TypeNode{sVar.ExplicitType}, SymbolVariable)

	// Lookup s.name in this scope
	sNameVar := ast.VariableNode{
		Ident: ast.Ident{ID: ast.Identifier("s.name")},
	}
	lookedUpType, err := tc.LookupVariableType(&sNameVar, scope)
	if err != nil {
		t.Errorf("unexpected error looking up s.name: %v", err)
	}
	if lookedUpType.Ident != ast.TypeString {
		t.Errorf("expected s.name to be String, got %s", lookedUpType.Ident)
	}
}

func TestShapeRefinementWithArgLookup(t *testing.T) {
	tc := New(logrus.New(), false)

	// Create base shape type
	baseShapeType := ast.TypeIdent("BaseShape")
	baseShape := ast.TypeDefNode{
		Ident: baseShapeType,
		Expr: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: typeIdentPtr(string(ast.TypeShape)),
			},
		},
	}
	tc.Defs[baseShapeType] = baseShape

	// Create a type guard that refines the shape with a shape argument
	guard := ast.TypeGuardNode{
		Ident: ast.Identifier("WithPerimeter"),
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: ast.Identifier("s")},
			Type:  ast.TypeNode{Ident: baseShapeType},
		},
		Params: []ast.ParamNode{
			ast.SimpleParamNode{
				Ident: ast.Ident{ID: ast.Identifier("perimeter")},
				Type:  ast.TypeNode{Ident: ast.TypeShape},
			},
		},
		Body: []ast.Node{
			ast.EnsureNode{
				Variable: ast.VariableNode{
					Ident: ast.Ident{ID: ast.Identifier("s")},
				},
				Assertion: ast.AssertionNode{
					BaseType: typeIdentPtr(string(ast.TypeShape)),
					Constraints: []ast.ConstraintNode{
						{
							Name: "HasField",
							Args: []ast.ConstraintArgumentNode{
								{
									Value: func() *ast.ValueNode {
										v := ast.StringLiteralNode{Value: "perimeter"}
										var n ast.ValueNode = v
										return &n
									}(),
								},
							},
						},
					},
				},
			},
		},
	}

	// Register the type guard in the global scope
	globalScope := tc.scopeStack.globalScope()
	globalScope.RegisterSymbol(guard.Ident, []ast.TypeNode{{Ident: ast.TypeVoid}}, SymbolTypeGuard)

	// Also register it in Defs for type lookup
	tc.Defs[ast.TypeIdent(guard.Ident)] = guard

	// Create a type alias that uses the type guard with a shape argument
	refinedShapeType := ast.TypeIdent("RefinedShape")
	refinedShape := ast.TypeDefNode{
		Ident: refinedShapeType,
		Expr: ast.TypeDefAssertionExpr{
			Assertion: &ast.AssertionNode{
				BaseType: &baseShapeType,
				Constraints: []ast.ConstraintNode{
					{
						Name: string(guard.Ident),
						Args: []ast.ConstraintArgumentNode{
							{
								Value: func() *ast.ValueNode {
									shape := ast.ShapeNode{
										Fields: map[string]ast.ShapeFieldNode{
											"width": {
												Type: &ast.TypeNode{Ident: ast.TypeInt},
											},
											"height": {
												Type: &ast.TypeNode{Ident: ast.TypeInt},
											},
										},
									}
									var n ast.ValueNode = shape
									return &n
								}(),
							},
						},
					},
				},
			},
		},
	}
	tc.Defs[refinedShapeType] = refinedShape

	// Register the shape type with the merged fields
	shapeType := ast.TypeDefNode{
		Ident: refinedShapeType,
		Expr: ast.TypeDefShapeExpr{
			Shape: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"perimeter": {
						Type: &ast.TypeNode{Ident: ast.TypeShape},
					},
					"type": {
						Type: &ast.TypeNode{Ident: ast.TypeString},
					},
				},
			},
		},
	}
	tc.Defs[refinedShapeType] = shapeType

	// Simulate a function scope and declare variable s of type RefinedShape
	scope := NewScope(nil, nil, logrus.New())
	sVar := ast.VariableNode{
		Ident:        ast.Ident{ID: ast.Identifier("s")},
		ExplicitType: ast.TypeNode{Ident: refinedShapeType},
	}
	scope.RegisterSymbol(sVar.Ident.ID, []ast.TypeNode{sVar.ExplicitType}, SymbolVariable)

	// Test accessing fields from both the base shape and the merged shape
	tests := []struct {
		name     string
		field    string
		expected ast.TypeIdent
	}{
		{"access perimeter", "s.perimeter", ast.TypeShape},
		{"access type", "s.type", ast.TypeString},
		{"access perimeter.width", "s.perimeter.width", ast.TypeShape},
		{"access perimeter.height", "s.perimeter.height", ast.TypeShape},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldVar := ast.VariableNode{
				Ident: ast.Ident{ID: ast.Identifier(tt.field)},
			}
			lookedUpType, err := tc.LookupVariableType(&fieldVar, scope)
			if err != nil {
				t.Errorf("unexpected error looking up %s: %v", tt.field, err)
			}
			if lookedUpType.Ident != tt.expected {
				t.Errorf("expected %s to be %s, got %s", tt.field, tt.expected, lookedUpType.Ident)
			}
		})
	}
}
