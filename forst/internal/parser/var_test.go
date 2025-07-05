package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParseVarStatement(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, assignment ast.AssignmentNode)
	}{
		{
			name: "var declaration with type and initializer",
			tokens: []ast.Token{
				{Type: ast.TokenVar, Value: "var", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "x", Line: 1, Column: 5},
				{Type: ast.TokenColon, Value: ":", Line: 1, Column: 7},
				{Type: ast.TokenInt, Value: "Int", Line: 1, Column: 9},
				{Type: ast.TokenEquals, Value: "=", Line: 1, Column: 13},
				{Type: ast.TokenIntLiteral, Value: "42", Line: 1, Column: 15},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 17},
			},
			validate: func(t *testing.T, assignment ast.AssignmentNode) {
				if len(assignment.LValues) != 1 {
					t.Fatalf("Expected 1 lvalue, got %d", len(assignment.LValues))
				}
				if assignment.LValues[0].Ident.ID != "x" {
					t.Errorf("Expected variable name 'x', got %s", assignment.LValues[0].Ident.ID)
				}
				if len(assignment.ExplicitTypes) != 1 {
					t.Fatalf("Expected 1 explicit type, got %d", len(assignment.ExplicitTypes))
				}
				if assignment.ExplicitTypes[0].Ident != ast.TypeInt {
					t.Errorf("Expected type 'int', got %s", assignment.ExplicitTypes[0].Ident)
				}
				if len(assignment.RValues) != 1 {
					t.Fatalf("Expected 1 rvalue, got %d", len(assignment.RValues))
				}
				literal := assertNodeType[ast.IntLiteralNode](t, assignment.RValues[0], "ast.IntLiteralNode")
				if literal.Value != 42 {
					t.Errorf("Expected value 42, got %d", literal.Value)
				}
				if assignment.IsShort {
					t.Error("Expected IsShort to be false for var declaration")
				}
			},
		},
		{
			name: "var declaration with type only",
			tokens: []ast.Token{
				{Type: ast.TokenVar, Value: "var", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "y", Line: 1, Column: 5},
				{Type: ast.TokenColon, Value: ":", Line: 1, Column: 7},
				{Type: ast.TokenString, Value: "String", Line: 1, Column: 9},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 15},
			},
			validate: func(t *testing.T, assignment ast.AssignmentNode) {
				if len(assignment.LValues) != 1 {
					t.Fatalf("Expected 1 lvalue, got %d", len(assignment.LValues))
				}
				if assignment.LValues[0].Ident.ID != "y" {
					t.Errorf("Expected variable name 'y', got %s", assignment.LValues[0].Ident.ID)
				}
				if len(assignment.ExplicitTypes) != 1 {
					t.Fatalf("Expected 1 explicit type, got %d", len(assignment.ExplicitTypes))
				}
				if assignment.ExplicitTypes[0].Ident != ast.TypeString {
					t.Errorf("Expected type 'string', got %s", assignment.ExplicitTypes[0].Ident)
				}
				if len(assignment.RValues) != 0 {
					t.Fatalf("Expected 0 rvalues, got %d", len(assignment.RValues))
				}
				if assignment.IsShort {
					t.Error("Expected IsShort to be false for var declaration")
				}
			},
		},
		{
			name: "var declaration without colon",
			tokens: []ast.Token{
				{Type: ast.TokenVar, Value: "var", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "z", Line: 1, Column: 5},
				{Type: ast.TokenBool, Value: "Bool", Line: 1, Column: 7},
				{Type: ast.TokenEquals, Value: "=", Line: 1, Column: 12},
				{Type: ast.TokenTrue, Value: "true", Line: 1, Column: 14},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 18},
			},
			validate: func(t *testing.T, assignment ast.AssignmentNode) {
				if len(assignment.LValues) != 1 {
					t.Fatalf("Expected 1 lvalue, got %d", len(assignment.LValues))
				}
				if assignment.LValues[0].Ident.ID != "z" {
					t.Errorf("Expected variable name 'z', got %s", assignment.LValues[0].Ident.ID)
				}
				if len(assignment.ExplicitTypes) != 1 {
					t.Fatalf("Expected 1 explicit type, got %d", len(assignment.ExplicitTypes))
				}
				if assignment.ExplicitTypes[0].Ident != ast.TypeBool {
					t.Errorf("Expected type 'bool', got %s", assignment.ExplicitTypes[0].Ident)
				}
				if len(assignment.RValues) != 1 {
					t.Fatalf("Expected 1 rvalue, got %d", len(assignment.RValues))
				}
				literal := assertNodeType[ast.BoolLiteralNode](t, assignment.RValues[0], "ast.BoolLiteralNode")
				if !literal.Value {
					t.Error("Expected value true, got false")
				}
				if assignment.IsShort {
					t.Error("Expected IsShort to be false for var declaration")
				}
			},
		},
		{
			name: "var declaration with slice type",
			tokens: []ast.Token{
				{Type: ast.TokenVar, Value: "var", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "arr", Line: 1, Column: 5},
				{Type: ast.TokenColon, Value: ":", Line: 1, Column: 9},
				{Type: ast.TokenLBracket, Value: "[", Line: 1, Column: 11},
				{Type: ast.TokenRBracket, Value: "]", Line: 1, Column: 12},
				{Type: ast.TokenInt, Value: "Int", Line: 1, Column: 14},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 17},
			},
			validate: func(t *testing.T, assignment ast.AssignmentNode) {
				if len(assignment.LValues) != 1 {
					t.Fatalf("Expected 1 lvalue, got %d", len(assignment.LValues))
				}
				if assignment.LValues[0].Ident.ID != "arr" {
					t.Errorf("Expected variable name 'arr', got %s", assignment.LValues[0].Ident.ID)
				}
				if len(assignment.ExplicitTypes) != 1 {
					t.Fatalf("Expected 1 explicit type, got %d", len(assignment.ExplicitTypes))
				}
				if assignment.ExplicitTypes[0].Ident != ast.TypeArray {
					t.Errorf("Expected type 'array', got %s", assignment.ExplicitTypes[0].Ident)
				}
				if len(assignment.ExplicitTypes[0].TypeParams) != 1 {
					t.Fatalf("Expected 1 type parameter, got %d", len(assignment.ExplicitTypes[0].TypeParams))
				}
				if assignment.ExplicitTypes[0].TypeParams[0].Ident != ast.TypeInt {
					t.Errorf("Expected element type 'int', got %s", assignment.ExplicitTypes[0].TypeParams[0].Ident)
				}
				if assignment.IsShort {
					t.Error("Expected IsShort to be false for var declaration")
				}
			},
		},
		{
			name: "var declaration with pointer type",
			tokens: []ast.Token{
				{Type: ast.TokenVar, Value: "var", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "ptr", Line: 1, Column: 5},
				{Type: ast.TokenColon, Value: ":", Line: 1, Column: 9},
				{Type: ast.TokenStar, Value: "*", Line: 1, Column: 11},
				{Type: ast.TokenString, Value: "String", Line: 1, Column: 12},
				{Type: ast.TokenEquals, Value: "=", Line: 1, Column: 18},
				{Type: ast.TokenNil, Value: "nil", Line: 1, Column: 20},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 23},
			},
			validate: func(t *testing.T, assignment ast.AssignmentNode) {
				if len(assignment.LValues) != 1 {
					t.Fatalf("Expected 1 lvalue, got %d", len(assignment.LValues))
				}
				if assignment.LValues[0].Ident.ID != "ptr" {
					t.Errorf("Expected variable name 'ptr', got %s", assignment.LValues[0].Ident.ID)
				}
				if len(assignment.ExplicitTypes) != 1 {
					t.Fatalf("Expected 1 explicit type, got %d", len(assignment.ExplicitTypes))
				}
				if assignment.ExplicitTypes[0].Ident != ast.TypePointer {
					t.Errorf("Expected type 'pointer', got %s", assignment.ExplicitTypes[0].Ident)
				}
				if len(assignment.ExplicitTypes[0].TypeParams) != 1 {
					t.Fatalf("Expected 1 type parameter, got %d", len(assignment.ExplicitTypes[0].TypeParams))
				}
				if assignment.ExplicitTypes[0].TypeParams[0].Ident != ast.TypeString {
					t.Errorf("Expected base type 'string', got %s", assignment.ExplicitTypes[0].TypeParams[0].Ident)
				}
				if len(assignment.RValues) != 1 {
					t.Fatalf("Expected 1 rvalue, got %d", len(assignment.RValues))
				}
				_, isNil := assignment.RValues[0].(ast.NilLiteralNode)
				if !isNil {
					t.Error("Expected nil literal")
				}
				if assignment.IsShort {
					t.Error("Expected IsShort to be false for var declaration")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := setupParser(tt.tokens)
			assignment := p.parseVarStatement()
			tt.validate(t, assignment)
		})
	}
}
