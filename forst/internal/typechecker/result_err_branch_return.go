package typechecker

import (
	"fmt"

	"forst/internal/ast"
)

// ifConditionIsBuiltinResultErrNarrowing is true when the `if` condition is `subject is Err(...)`
// on a built-in Result(S, F) value (see refinedTypesForResultIsNarrowing). The then-branch is the
// Result failure region for ensure-only failure propagation (errors RFC 01).
func (tc *TypeChecker) ifConditionIsBuiltinResultErrNarrowing(condition ast.Node) bool {
	if condition == nil {
		return false
	}
	bin, ok := condition.(ast.BinaryExpressionNode)
	if !ok || bin.Operator != ast.TokenIs {
		return false
	}
	leftmostVar, err := tc.getLeftmostVariable(bin.Left)
	if err != nil {
		return false
	}
	varLeftTypes, err := tc.inferExpressionType(leftmostVar)
	if err != nil || len(varLeftTypes) != 1 {
		return false
	}
	a := assertionNodeFromIsRHS(bin.Right)
	if a == nil {
		return false
	}
	handled, _, err := tc.refinedTypesForResultIsNarrowing(varLeftTypes[0], a)
	if !handled || err != nil {
		return false
	}
	if len(a.Constraints) == 0 {
		return false
	}
	return a.Constraints[0].Name == "Err"
}

// checkReturnDisallowedInResultErrBranch rejects `return Err(...)` inside the then-branch of
// `if x is Err(...)` on Result — propagate with `ensure x is Ok()` (or `ensure ... or err`) instead.
func (tc *TypeChecker) checkReturnDisallowedInResultErrBranch(ret ast.ReturnNode) error {
	if tc.resultErrIfBranchDepth == 0 {
		return nil
	}
	if tc.currentFunction == nil || len(tc.currentFunction.ReturnTypes) != 1 {
		return nil
	}
	rt := tc.currentFunction.ReturnTypes[0]
	if !rt.IsResultType() || len(rt.TypeParams) < 2 {
		return nil
	}
	for _, expr := range ret.Values {
		if isErrExprAST(expr) {
			return fmt.Errorf("propagate Result failures with `ensure` (e.g. `ensure x is Ok()`), not `if` + `return Err(...)` in an `Err` branch")
		}
	}
	return nil
}

func isErrExprAST(expr ast.ExpressionNode) bool {
	switch e := expr.(type) {
	case ast.ErrExprNode:
		return true
	case *ast.ErrExprNode:
		return e != nil
	default:
		return false
	}
}
