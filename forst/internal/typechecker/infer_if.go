package typechecker

import (
	"forst/internal/ast"
)

// inferIfStatement type-checks if / else-if / else, including init and conditions.
//
// Control-flow narrowing (design notes — see repo narrowing plan §1):
//   - TypeScript/Flow-style flow-sensitive facts: branch-local refinement for `x is R` via
//     lexical scopes (pushScope) + RegisterSymbolWithNarrowing in narrow_if.go.
//   - SSA/phi analogy: endIfChainApplyJoin is the join at the merge point after the if-chain;
//     JoinAfterIfMerge (typeops.go) widens to the outer (pre-if) binding type (§3.2) until unions exist.
//   - Kotlin/Swift analogy: refinement is scoped to branch bodies; after the full chain, merge
//     restores the enclosing binding type for uses in the continuation.
func (tc *TypeChecker) inferIfStatement(n ast.IfNode) ([]ast.TypeNode, error) {
	tc.beginIfChainForStatement()
	defer tc.endIfChainApplyJoin()

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
	incErr := tc.ifConditionIsBuiltinResultErrNarrowing(n.Condition)
	tc.pushScope(&n)
	if incErr {
		tc.resultErrIfBranchDepth++
	}
	// Lexical shadowing: `if x is Assertion { ... }` refines x inside the then-branch.
	tc.applyIfBranchNarrowing(n.Condition)
	for _, node := range n.Body {
		if _, err := tc.inferNodeType(node); err != nil {
			if incErr {
				tc.resultErrIfBranchDepth--
			}
			tc.popScope()
			return nil, err
		}
	}
	if incErr {
		tc.resultErrIfBranchDepth--
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
		incErrEI := tc.ifConditionIsBuiltinResultErrNarrowing(ei.Condition)
		tc.pushScope(&ei)
		if incErrEI {
			tc.resultErrIfBranchDepth++
		}
		tc.applyIfBranchNarrowing(ei.Condition)
		for _, node := range ei.Body {
			if _, err := tc.inferNodeType(node); err != nil {
				if incErrEI {
					tc.resultErrIfBranchDepth--
				}
				tc.popScope()
				return nil, err
			}
		}
		if incErrEI {
			tc.resultErrIfBranchDepth--
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
