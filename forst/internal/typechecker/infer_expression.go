package typechecker

import (
	"fmt"
	"forst/internal/ast"
	"strings"

	log "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) inferExpressionType(expr ast.Node) ([]ast.TypeNode, error) {
	log.Tracef("inferExpressionType: %T", expr)
	switch e := expr.(type) {
	case ast.BinaryExpressionNode:
		inferredType, err := tc.unifyTypes(e.Left, e.Right, e.Operator)
		if err != nil {
			return nil, err
		}
		tc.storeInferredType(e, []ast.TypeNode{inferredType})
		return []ast.TypeNode{inferredType}, nil

	case ast.UnaryExpressionNode:
		inferredType, err := tc.unifyTypes(e.Operand, nil, e.Operator)
		if err != nil {
			return nil, err
		}
		tc.storeInferredType(e, []ast.TypeNode{inferredType})
		return []ast.TypeNode{inferredType}, nil

	case ast.IntLiteralNode:
		typ := ast.TypeNode{Ident: ast.TypeInt}
		tc.storeInferredType(e, []ast.TypeNode{typ})
		return []ast.TypeNode{typ}, nil

	case ast.FloatLiteralNode:
		typ := ast.TypeNode{Ident: ast.TypeFloat}
		tc.storeInferredType(e, []ast.TypeNode{typ})
		return []ast.TypeNode{typ}, nil

	case ast.StringLiteralNode:
		typ := ast.TypeNode{Ident: ast.TypeString}
		tc.storeInferredType(e, []ast.TypeNode{typ})
		return []ast.TypeNode{typ}, nil

	case ast.BoolLiteralNode:
		typ := ast.TypeNode{Ident: ast.TypeBool}
		tc.storeInferredType(e, []ast.TypeNode{typ})
		return []ast.TypeNode{typ}, nil

	case ast.VariableNode:
		// Look up the variable's type and store it for this node
		typ, err := tc.LookupVariableType(&e)
		if err != nil {
			return nil, err
		}
		tc.storeInferredType(e, []ast.TypeNode{typ})
		return []ast.TypeNode{typ}, nil

	case ast.FunctionCallNode:
		log.Tracef("Checking function call: %s with %d arguments", e.Function.ID, len(e.Arguments))
		if signature, exists := tc.Functions[e.Function.ID]; exists {
			log.Tracef("Found function signature for %s: %v", e.Function.ID, signature.ReturnTypes)
			tc.storeInferredType(e, signature.ReturnTypes)
			return signature.ReturnTypes, nil
		}
		// For type guards, we need to ensure they return boolean
		if typeGuard, exists := tc.scopeStack.GlobalScope().Symbols[e.Function.ID]; exists && typeGuard.Kind == SymbolFunction {
			log.Tracef("Found type guard %s with types: %v", e.Function.ID, typeGuard.Types)
			return typeGuard.Types, nil
		}

		// First check if this is a local variable or method call
		if varType, exists := tc.scopeStack.LookupVariableType(e.Function.ID); exists {
			log.Tracef("Found local variable %s with type: %v", e.Function.ID, varType)
			return varType, nil
		}

		// Then check if this is a package-qualified function call
		parts := strings.Split(string(e.Function.ID), ".")
		if len(parts) == 2 {
			pkgName := parts[0]
			funcName := parts[1]

			// First check if pkgName is a local variable
			if varType, exists := tc.scopeStack.LookupVariableType(ast.Identifier(pkgName)); exists {
				log.Tracef("Found local variable %s with type: %v", pkgName, varType)
				// Check if the method is valid for this type
				returnType, err := tc.inferMethodCallType(varType, funcName, e.Arguments)
				if err != nil {
					return nil, err
				}
				tc.storeInferredType(e, returnType)
				return returnType, nil
			}

			// If not a local variable, check for built-in functions
			qualifiedName := pkgName + "." + funcName
			if builtin, exists := BuiltinFunctions[qualifiedName]; exists {
				log.Tracef("Found built-in function %s", qualifiedName)
				returnType, err := tc.checkBuiltinFunctionCall(builtin, e.Arguments)
				if err != nil {
					return nil, err
				}
				tc.storeInferredType(e, returnType)
				return returnType, nil
			}
		} else {
			// Check for unqualified built-in functions (like len)
			if builtin, exists := BuiltinFunctions[string(e.Function.ID)]; exists {
				log.Tracef("Found built-in function %s", e.Function.ID)
				returnType, err := tc.checkBuiltinFunctionCall(builtin, e.Arguments)
				if err != nil {
					return nil, err
				}
				tc.storeInferredType(e, returnType)
				return returnType, nil
			}
		}

		log.Tracef("No function found for %s", e.Function.ID)
		return nil, fmt.Errorf("unknown identifier: %s", e.Function.ID)

	default:
		log.Tracef("Unhandled expression type: %T", expr)
		return nil, fmt.Errorf("cannot infer type for expression: %T", expr)
	}
}

// Checks if a method call is valid for a given type and returns its return type
func (tc *TypeChecker) inferMethodCallType(varType []ast.TypeNode, methodName string, args []ast.ExpressionNode) ([]ast.TypeNode, error) {
	log.Tracef("inferMethodCallType: varType=%v, methodName=%s, args=%v", varType, methodName, args)

	if len(varType) != 1 {
		log.Tracef("Method calls are only valid on single types, got %s", formatTypeList(varType))
		return nil, fmt.Errorf("method calls are only valid on single types, got %s", formatTypeList(varType))
	}

	returnType, err := CheckBuiltinMethod(varType[0], methodName, args)
	if err != nil {
		log.Tracef("Error checking built-in method: %v", err)
		return nil, err
	}

	log.Tracef("Successfully inferred method call type: %v", returnType)
	return returnType, nil
}
