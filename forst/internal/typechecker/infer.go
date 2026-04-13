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
		if err := tc.RestoreScope(scopeNode); err != nil {
			return nil, err
		}
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
		return tc.inferFunctionNode(n)

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
		return tc.inferEnsureNode(n)
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

			// Assertion-only alias with no shape fields (e.g. type MyStr = String via TypeDefAssertionExpr,
			// or a chain of such aliases): do not replace Defs with an empty TypeDefShapeExpr — the collect
			// pass already stored TypeDefAssertionExpr and alias-chain / narrowing logic needs it.
			if len(mergedFields) == 0 && len(assertionExpr.Assertion.Constraints) == 0 &&
				assertionExpr.Assertion.BaseType != nil {
				base := *assertionExpr.Assertion.BaseType
				// Direct alias to a built-in (e.g. type Greeting = String): underlyingBuiltinTypeOfAliasAssertion
				// only resolves named typedef chains via Defs and returns "" when base is itself a built-in ident.
				if tc.isBuiltinType(base) {
					return nil, nil
				}
				if tc.underlyingBuiltinTypeOfAliasAssertion(base) != "" {
					return nil, nil
				}
			}

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
		// Infer return value expressions in this scope (including nested returns under if/ensure)
		// so per-occurrence types and narrowing metadata (e.g. type guard names) are stored.
		for _, v := range n.Values {
			if _, err := tc.inferExpressionType(v); err != nil {
				return nil, err
			}
		}
		if err := tc.checkReturnDisallowedInResultErrBranch(n); err != nil {
			return nil, err
		}
		return nil, nil

	case ast.ImportNode:
		return nil, nil

	case ast.ImportGroupNode:
		return nil, nil

	case ast.TypeGuardNode, *ast.TypeGuardNode:
		return tc.inferTypeGuardNode(n)

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

	case *ast.DeferNode:
		fc, ok := n.Call.(ast.FunctionCallNode)
		if !ok {
			return nil, fmt.Errorf("defer requires a function or method call")
		}
		if err := validateDeferGoBuiltinRestriction("defer", fc); err != nil {
			return nil, err
		}
		if _, err := tc.inferExpressionType(n.Call); err != nil {
			return nil, err
		}
		return nil, nil
	case *ast.GoStmtNode:
		fc, ok := n.Call.(ast.FunctionCallNode)
		if !ok {
			return nil, fmt.Errorf("go requires a function or method call")
		}
		if err := validateDeferGoBuiltinRestriction("go", fc); err != nil {
			return nil, err
		}
		if _, err := tc.inferExpressionType(n.Call); err != nil {
			return nil, err
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

	case ast.CommentNode:
		return nil, nil
	}

	return nil, fmt.Errorf("%s", typecheckErrorMessageWithNode(&node, fmt.Sprintf("unsupported node type %T", node)))
}
