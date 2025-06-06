package typechecker

import (
	"fmt"
	"forst/internal/ast"

	log "github.com/sirupsen/logrus"
)

// inferNodeTypes handles type inference for a list of nodes
func (tc *TypeChecker) inferNodeTypes(nodes []ast.Node) ([][]ast.TypeNode, error) {
	log.Trace("inferNodeTypes", nodes)
	inferredTypes := make([][]ast.TypeNode, len(nodes))
	for i, node := range nodes {
		inferredType, err := tc.inferNodeType(node)
		if err != nil {
			return nil, err
		}

		inferredTypes[i] = inferredType
	}
	return inferredTypes, nil
}

// inferNodeType handles type inference for a single node
func (tc *TypeChecker) inferNodeType(node ast.Node) ([]ast.TypeNode, error) {
	log.Tracef("inferNodeType: %s", node.String())

	switch n := node.(type) {
	case ast.PackageNode:
		return nil, nil
	case ast.FunctionNode:
		tc.pushScope(n)

		// Register parameters in the current scope
		for _, param := range n.Params {
			switch p := param.(type) {
			case ast.SimpleParamNode:
				tc.scopeStack.CurrentScope().DefineVariable(p.Ident.ID, p.Type)
			case ast.DestructuredParamNode:
				// Handle destructured params if needed
				continue
			}
		}

		// Convert []ParamNode to []Node
		params := make([]ast.Node, len(n.Params))
		for i, param := range n.Params {
			params[i] = param
		}

		paramTypes, err := tc.inferNodeTypes(params)
		if err != nil {
			return nil, err
		}

		for i, paramTypes := range paramTypes {
			param := n.Params[i]
			// Store in scope for structural lookup
			log.Tracef("inferNodeType: storing param variable type %v for %s", paramTypes, param.GetIdent())
			tc.scopeStack.CurrentScope().DefineVariable(ast.Identifier(param.GetIdent()), paramTypes[0])
		}

		_, err = tc.inferNodeTypes(n.Body)
		if err != nil {
			return nil, err
		}

		inferredType, err := tc.inferFunctionReturnType(n)
		if err != nil {
			return nil, err
		}
		tc.storeInferredFunctionReturnType(&n, inferredType)

		tc.popScope()

		return inferredType, nil

	case ast.SimpleParamNode:
		if n.Type.Assertion != nil {
			inferredType, err := tc.inferAssertionType(n.Type.Assertion, false)
			if err != nil {
				return nil, err
			}
			return inferredType, nil
		}
		return []ast.TypeNode{n.Type}, nil

	case ast.DestructuredParamNode:
		return nil, nil

	case ast.ExpressionNode:
		inferredType, err := tc.inferExpressionType(n)
		if err != nil {
			return nil, err
		}
		tc.storeInferredType(n, inferredType)
		return inferredType, nil

	case ast.EnsureNode:
		_, err := tc.inferEnsureType(n)
		if err != nil {
			return nil, err
		}

		if n.Block != nil {
			tc.pushScope(n.Block)
			_, err = tc.inferNodeTypes(n.Block.Body)
			if err != nil {
				return nil, err
			}
			tc.popScope()
		}

		return nil, nil
	case ast.AssignmentNode:
		if err := tc.inferAssignmentTypes(n); err != nil {
			return nil, err
		}
		return nil, nil

	case ast.TypeNode:
		return nil, nil

	case ast.TypeDefNode:
		if assertionExpr, ok := n.Expr.(ast.TypeDefAssertionExpr); ok && assertionExpr.Assertion != nil {
			log.Debugf("[TypeDefNode] Merging fields for type %s", n.Ident)
			mergedFields := tc.resolveMergedShapeFields(assertionExpr.Assertion)
			log.Debugf("[TypeDefNode] Merged fields for %s: %v", n.Ident, mergedFields)
			shape := ast.ShapeNode{
				Fields: mergedFields,
			}
			log.Debugf("[TypeDefNode] Registering merged shape for %s: %+v", n.Ident, shape)
			tc.registerShapeType(n.Ident, shape)
		}
		return nil, nil

	case ast.ReturnNode:
		return nil, nil

	case ast.ImportNode:
		return nil, nil

	case ast.ImportGroupNode:
		return nil, nil

	case *ast.TypeGuardNode:
		// Push a new scope for the type guard's body
		tc.pushScope(n)

		// Register parameters in the current scope
		for _, param := range n.Parameters() {
			switch p := param.(type) {
			case ast.SimpleParamNode:
				tc.scopeStack.CurrentScope().DefineVariable(p.Ident.ID, p.Type)
			case ast.DestructuredParamNode:
				continue
			}
		}

		// Validate type guard body
		for _, node := range n.Body {
			switch stmt := node.(type) {
			case ast.IfNode:
				// Check that condition uses is operator
				if binExpr, ok := stmt.Condition.(ast.BinaryExpressionNode); !ok || binExpr.Operator != ast.TokenIs {
					return nil, fmt.Errorf("type guard conditions must use 'is' operator")
				}
			case ast.EnsureNode:
				// Ensure statements are valid
			case ast.ReturnNode:
				// Return statements are not allowed in type guards
				return nil, fmt.Errorf("type guards must not have return statements")
			default:
				// Only if, else if, else, and ensure statements are allowed
				return nil, fmt.Errorf("type guards may only contain if, else if, else, and ensure statements")
			}
		}

		// Store type guard in global scope with void return type
		tc.storeSymbol(ast.Identifier(n.Ident), []ast.TypeNode{{Ident: ast.TypeVoid}}, SymbolTypeGuard)

		tc.popScope()
		return nil, nil
	}

	panic(typecheckErrorMessageWithNode(&node, fmt.Sprintf("unsupported node type %T", node)))
}
