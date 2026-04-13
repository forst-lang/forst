package typechecker

import (
	"fmt"
	"forst/internal/ast"
)

func (tc *TypeChecker) inferTypeGuardNode(typeGuardNode ast.Node) ([]ast.TypeNode, error) {
	var guardNode ast.TypeGuardNode
	if ptr, ok := typeGuardNode.(*ast.TypeGuardNode); ok {
		guardNode = *ptr
	} else {
		guardNode = typeGuardNode.(ast.TypeGuardNode)
	}
	tc.pushScope(&guardNode)

	for _, param := range guardNode.Parameters() {
		switch typedParam := param.(type) {
		case ast.SimpleParamNode:
			parameterTypes := []ast.TypeNode{typedParam.Type}
			tc.scopeStack.currentScope().RegisterSymbol(
				typedParam.Ident.ID,
				parameterTypes,
				SymbolVariable)
			tc.VariableTypes[typedParam.Ident.ID] = parameterTypes
		case ast.DestructuredParamNode:
			continue
		}
	}

	for _, node := range guardNode.Body {
		switch stmt := node.(type) {
		case ast.CommentNode:
			continue
		case ast.ReturnNode:
			return nil, fmt.Errorf("type guards must not have return statements")
		case ast.IfNode:
			if binExpr, ok := stmt.Condition.(ast.BinaryExpressionNode); !ok || binExpr.Operator != ast.TokenIs {
				return nil, fmt.Errorf("type guard conditions must use 'is' operator")
			}
			if _, err := tc.inferIfStatement(stmt); err != nil {
				return nil, err
			}
		case *ast.IfNode:
			if binExpr, ok := stmt.Condition.(ast.BinaryExpressionNode); !ok || binExpr.Operator != ast.TokenIs {
				return nil, fmt.Errorf("type guard conditions must use 'is' operator")
			}
			if _, err := tc.inferIfStatement(*stmt); err != nil {
				return nil, err
			}
		case ast.EnsureNode:
			if _, err := tc.inferNodeType(stmt); err != nil {
				return nil, err
			}
		case *ast.EnsureNode:
			if _, err := tc.inferNodeType(*stmt); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("type guards may only contain if, else if, else, and ensure statements")
		}
	}

	tc.popScope()
	return nil, nil
}
