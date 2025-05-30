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
	// Check if we've already inferred this node's type
	alreadyInferredType, err := tc.LookupInferredType(node, false)
	if err != nil {
		return nil, err
	}
	if alreadyInferredType != nil {
		// fmt.Println("inferNodeType", node, "already inferred", alreadyInferredType)
		// return alreadyInferredType, nil
	}

	switch n := node.(type) {
	case ast.PackageNode:
		return nil, nil
	case ast.FunctionNode:
		tc.pushScope(n)

		// Register parameters in the current scope
		for _, param := range n.Params {
			switch p := param.(type) {
			case ast.SimpleParamNode:
				tc.scopeStack.CurrentScope().Symbols[p.Ident.Id] = Symbol{
					Identifier: p.Ident.Id,
					Types:      []ast.TypeNode{p.Type},
					Kind:       SymbolVariable,
					Scope:      tc.scopeStack.CurrentScope(),
					Position:   tc.path,
				}
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

		_, err = tc.inferNodeTypes(params)
		if err != nil {
			return nil, err
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

	case ast.AssertionNode:
		_, err := tc.inferAssertionType(&n, false)
		if err != nil {
			return nil, err
		}
		return nil, nil

	case ast.TypeNode:
		return nil, nil

	case ast.TypeDefNode:
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
				tc.scopeStack.CurrentScope().Symbols[p.Ident.Id] = Symbol{
					Identifier: p.Ident.Id,
					Types:      []ast.TypeNode{p.Type},
					Kind:       SymbolVariable,
					Scope:      tc.scopeStack.CurrentScope(),
					Position:   tc.path,
				}
			case ast.DestructuredParamNode:
				// Handle destructured params if needed
				continue
			}
		}

		// Type guards must return a boolean value
		if len(n.Body) == 0 {
			tc.popScope()
			return nil, fmt.Errorf("type guard must have a return statement")
		}
		returnNode, ok := n.Body[0].(ast.ReturnNode)
		if !ok {
			tc.popScope()
			return nil, fmt.Errorf("type guard must have a return statement")
		}
		returnType, err := tc.inferExpressionType(returnNode.Value)
		if err != nil {
			tc.popScope()
			return nil, err
		}
		if len(returnType) != 1 || returnType[0].Ident != ast.TypeBool {
			tc.popScope()
			return nil, fmt.Errorf("type guard must return a boolean value, got %s", formatTypeList(returnType))
		}

		// Store type guard in global scope with the inferred return type
		tc.storeSymbol(ast.Identifier(n.Ident), returnType, SymbolFunction)

		// Store the inferred type for the type guard itself
		tc.storeInferredType(n, returnType)

		// Store the type guard in the Functions map for easier lookup
		tc.Functions[n.Ident] = FunctionSignature{
			Ident:       ast.Ident{Id: n.Ident},
			ReturnTypes: returnType,
		}

		tc.popScope()
		return returnType, nil
	}

	panic(typecheckErrorMessageWithNode(&node, fmt.Sprintf("unsupported node type %T", node)))
}
