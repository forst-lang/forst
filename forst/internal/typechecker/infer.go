package typechecker

import (
	"fmt"
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

// inferNodeTypes handles type inference for a list of nodes
func (tc *TypeChecker) inferNodeTypes(nodes []ast.Node, scopeNode ast.Node) ([][]ast.TypeNode, error) {
	inferredTypes := make([][]ast.TypeNode, len(nodes))
	for i, node := range nodes {
		tc.RestoreScope(scopeNode)
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
	tc.log.WithFields(logrus.Fields{
		"node":     node.String(),
		"function": "inferNodeType",
	}).Trace("Inferring node type")

	switch n := node.(type) {
	case ast.PackageNode:
		return nil, nil
	case ast.FunctionNode:
		tc.log.WithFields(logrus.Fields{
			"function": "inferNodeType",
			"fn":       n.Ident.ID,
			"phase":    "ENTER",
		}).Debug("Function node type inference")

		if err := tc.RestoreScope(n); err != nil {
			return nil, err
		}
		tc.log.WithFields(logrus.Fields{
			"function": "inferNodeType",
			"fn":       n.Ident.ID,
		}).Debug("Restored function scope")

		// Register parameters in the current scope
		for _, param := range n.Params {
			switch p := param.(type) {
			case ast.SimpleParamNode:
				tc.scopeStack.currentScope().RegisterSymbol(
					p.Ident.ID,
					[]ast.TypeNode{p.Type},
					SymbolVariable)
			case ast.DestructuredParamNode:
				continue
			}
		}
		tc.DebugPrintCurrentScope() // Print all symbols in the current scope after parameter registration

		// Convert []ParamNode to []Node
		params := make([]ast.Node, len(n.Params))
		for i, param := range n.Params {
			params[i] = param
		}

		paramTypes, err := tc.inferNodeTypes(params, n)
		if err != nil {
			return nil, err
		}

		for i, paramTypes := range paramTypes {
			param := n.Params[i]
			// Store in scope for structural lookup
			tc.log.WithFields(logrus.Fields{
				"paramTypes": paramTypes,
				"param":      param.GetIdent(),
				"function":   "inferNodeType",
			}).Trace("Storing param variable type")

			tc.scopeStack.currentScope().RegisterSymbol(
				ast.Identifier(param.GetIdent()),
				paramTypes,
				SymbolVariable)
		}

		// Process function body without restoring scope for each node
		// since we want to preserve the function parameter scope
		for _, bodyNode := range n.Body {
			_, err := tc.inferNodeType(bodyNode)
			if err != nil {
				return nil, err
			}
		}

		inferredType, err := tc.inferFunctionReturnType(n)
		if err != nil {
			return nil, err
		}
		tc.storeInferredFunctionReturnType(&n, inferredType)

		tc.popScope()

		tc.log.WithFields(logrus.Fields{
			"function": "inferNodeType",
			"fn":       n.Ident.ID,
			"phase":    "EXIT",
		}).Debug("Function node type inference")

		return inferredType, nil

	case ast.SimpleParamNode:
		if n.Type.Assertion != nil {
			inferredType, err := tc.InferAssertionType(n.Type.Assertion, false, "", nil)
			if err != nil {
				return nil, err
			}
			return inferredType, nil
		}
		return []ast.TypeNode{n.Type}, nil

	case ast.DestructuredParamNode:
		return nil, nil

	case ast.ExpressionNode:
		tc.log.WithFields(logrus.Fields{
			"function": "inferNodeType",
			"expr":     n.String(),
		}).Debug("Processing expression node")
		inferredType, err := tc.inferExpressionType(n)
		if err != nil {
			return nil, err
		}
		tc.storeInferredType(n, inferredType)
		return inferredType, nil

	case ast.EnsureNode:
		variableType, err := tc.inferEnsureType(n)
		if err != nil {
			return nil, err
		}

		if n.Block != nil {
			tc.pushScope(n.Block)
			tc.applyEnsureSuccessorNarrowing(n)
			_, err = tc.inferNodeTypes(n.Block.Body, n.Block)
			tc.popScope()
			if err != nil {
				return nil, err
			}
		} else {
			tc.applyEnsureSuccessorNarrowing(n)
		}

		tc.storeInferredType(n.Assertion, []ast.TypeNode{variableType})

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
			tc.log.WithFields(logrus.Fields{
				"ident":    n.Ident,
				"function": "inferNodeType",
			}).Debug("Merging fields for type")

			mergedFields := tc.resolveShapeFieldsFromAssertion(assertionExpr.Assertion)
			tc.log.WithFields(logrus.Fields{
				"ident":        n.Ident,
				"mergedFields": mergedFields,
			}).Debug("Merged fields for type")

			shape := ast.ShapeNode{
				Fields: mergedFields,
			}
			tc.log.WithFields(logrus.Fields{
				"ident":    n.Ident,
				"shape":    shape,
				"function": "inferNodeType",
			}).Debug("Registering merged shape for type")

			tc.registerShapeType(n.Ident, shape)
		}
		return nil, nil

	case ast.ReturnNode:
		// For return statements, we don't need to infer types here
		// as they are handled in inferFunctionReturnType
		return nil, nil

	case ast.ImportNode:
		return nil, nil

	case ast.ImportGroupNode:
		return nil, nil

	case ast.TypeGuardNode, *ast.TypeGuardNode:
		// Push a new scope for the type guard's body
		var guardNode ast.TypeGuardNode
		if ptr, ok := n.(*ast.TypeGuardNode); ok {
			guardNode = *ptr
		} else {
			guardNode = n.(ast.TypeGuardNode)
		}
		tc.pushScope(&guardNode)

		// Register parameters in the current scope
		for _, param := range guardNode.Parameters() {
			switch p := param.(type) {
			case ast.SimpleParamNode:
				tc.scopeStack.currentScope().RegisterSymbol(
					p.Ident.ID,
					[]ast.TypeNode{p.Type}, SymbolVariable)
			case ast.DestructuredParamNode:
				continue
			}
		}

		// Validate type guard body
		for _, node := range guardNode.Body {
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

		tc.popScope()
		return nil, nil

	case ast.IfNode:
		return tc.inferIfStatement(n)
	case *ast.IfNode:
		return tc.inferIfStatement(*n)

	case *ast.ForNode:
		return tc.inferForNode(n)
	case ast.ForNode:
		return tc.inferForNode(&n)

	case *ast.BreakNode:
		if n.Label != nil {
			return nil, fmt.Errorf("labeled break is not implemented yet")
		}
		if tc.loopDepth == 0 {
			return nil, fmt.Errorf("break is not inside a loop")
		}
		return nil, nil
	case *ast.ContinueNode:
		if n.Label != nil {
			return nil, fmt.Errorf("labeled continue is not implemented yet")
		}
		if tc.loopDepth == 0 {
			return nil, fmt.Errorf("continue is not inside a loop")
		}
		return nil, nil

	case *ast.ElseBlockNode:
		if n == nil {
			return nil, nil
		}
		return tc.inferNodeType(*n)
	case ast.ElseBlockNode:
		tc.pushScope(n)
		for _, node := range n.Body {
			if _, err := tc.inferNodeType(node); err != nil {
				return nil, err
			}
		}
		tc.popScope()
		return nil, nil
	}

	return nil, fmt.Errorf("%s", typecheckErrorMessageWithNode(&node, fmt.Sprintf("unsupported node type %T", node)))
}
