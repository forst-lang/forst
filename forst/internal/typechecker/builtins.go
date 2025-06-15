package typechecker

import (
	"fmt"
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

// BuiltinType represents a built-in type and its available methods
type BuiltinType struct {
	Methods map[string]BuiltinMethod
}

// BuiltinMethod represents a method available on a built-in type
type BuiltinMethod struct {
	ReturnType ast.TypeNode
	ArgTypes   []ast.TypeNode
}

// BuiltinTypes maps type identifiers to their available methods
var BuiltinTypes = map[ast.TypeIdent]BuiltinType{
	ast.TypeError: {
		Methods: map[string]BuiltinMethod{
			"Error": {
				ReturnType: ast.TypeNode{Ident: ast.TypeString},
				ArgTypes:   []ast.TypeNode{},
			},
		},
	},
	ast.TypeString: {
		Methods: map[string]BuiltinMethod{
			"len": {
				ReturnType: ast.TypeNode{Ident: ast.TypeInt},
				ArgTypes:   []ast.TypeNode{},
			},
		},
	},
	ast.TypeInt: {
		Methods: map[string]BuiltinMethod{
			"toString": {
				ReturnType: ast.TypeNode{Ident: ast.TypeString},
				ArgTypes:   []ast.TypeNode{},
			},
		},
	},
	ast.TypeFloat: {
		Methods: map[string]BuiltinMethod{
			"toString": {
				ReturnType: ast.TypeNode{Ident: ast.TypeString},
				ArgTypes:   []ast.TypeNode{},
			},
		},
	},
	ast.TypeBool: {
		Methods: map[string]BuiltinMethod{
			"toString": {
				ReturnType: ast.TypeNode{Ident: ast.TypeString},
				ArgTypes:   []ast.TypeNode{},
			},
		},
	},
}

// CheckBuiltinMethod checks if a method call is valid for a built-in type and returns its return type
func (tc *TypeChecker) CheckBuiltinMethod(typ ast.TypeNode, methodName string, args []ast.ExpressionNode) ([]ast.TypeNode, error) {
	tc.log.WithFields(logrus.Fields{
		"function": "CheckBuiltinMethod",
		"method":   methodName,
		"type":     typ.Ident,
		"args":     args,
	}).Tracef("Checking built-in method")

	builtinType, exists := BuiltinTypes[typ.Ident]
	if !exists {
		tc.log.WithFields(logrus.Fields{
			"function": "CheckBuiltinMethod",
			"method":   methodName,
			"type":     typ.Ident,
			"args":     args,
		}).Tracef("Type is not a built-in type: %s", typ.Ident)
		return nil, fmt.Errorf("type %s is not a built-in type", typ.Ident)
	}

	method, exists := builtinType.Methods[methodName]
	if !exists {
		tc.log.WithFields(logrus.Fields{
			"function": "CheckBuiltinMethod",
			"method":   methodName,
			"type":     typ.Ident,
			"args":     args,
		}).Tracef("Method %s is not valid on type %s", methodName, typ.Ident)
		return nil, fmt.Errorf("method %s() is not valid on type %s", methodName, typ.Ident)
	}

	// Check argument count
	if len(args) != len(method.ArgTypes) {
		tc.log.WithFields(logrus.Fields{
			"function": "CheckBuiltinMethod",
			"method":   methodName,
			"type":     typ.Ident,
			"args":     args,
		}).Tracef("Method %s expects %d arguments, got %d", methodName, len(method.ArgTypes), len(args))
		return nil, fmt.Errorf("method %s() expects %d arguments, got %d", methodName, len(method.ArgTypes), len(args))
	}

	tc.log.WithFields(logrus.Fields{
		"function": "CheckBuiltinMethod",
		"method":   methodName,
		"type":     typ.Ident,
		"args":     args,
	}).Tracef("Method %s is valid, returning type %s", methodName, method.ReturnType.Ident)
	return []ast.TypeNode{method.ReturnType}, nil
}
