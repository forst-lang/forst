package typechecker

import (
	"forst/internal/ast"
)

// inferIfStatement type-checks if / else-if / else, including init and conditions.
func (tc *TypeChecker) inferIfStatement(n ast.IfNode) ([]ast.TypeNode, error) {
	if n.Init != nil {
		if _, err := tc.inferNodeType(n.Init); err != nil {
			return nil, err
		}
	}
	if n.Condition != nil {
		if ce, ok := n.Condition.(ast.ExpressionNode); ok {
			if _, err := tc.inferExpressionType(ce); err != nil {
				return nil, err
			}
		}
	}
	tc.pushScope(&n)
	// Lexical shadowing: `if x is Assertion { ... }` refines x inside the then-branch.
	tc.applyIfBranchNarrowing(n.Condition)
	for _, node := range n.Body {
		if _, err := tc.inferNodeType(node); err != nil {
			tc.popScope()
			return nil, err
		}
	}
	tc.popScope()

	for _, ei := range n.ElseIfs {
		if ei.Condition != nil {
			if ce, ok := ei.Condition.(ast.ExpressionNode); ok {
				if _, err := tc.inferExpressionType(ce); err != nil {
					return nil, err
				}
			}
		}
		tc.pushScope(&ei)
		tc.applyIfBranchNarrowing(ei.Condition)
		for _, node := range ei.Body {
			if _, err := tc.inferNodeType(node); err != nil {
				tc.popScope()
				return nil, err
			}
		}
		tc.popScope()
	}
	if n.Else != nil {
		if _, err := tc.inferNodeType(n.Else); err != nil {
			return nil, err
		}
	}
	return nil, nil
}
