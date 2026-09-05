package typechecker

import (
	"slices"

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
	handled, _, err := tc.refinedTypesForResultIsNarrowing(varLeftTypes[0], a, spanOfNode(leftmostVar))
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
	if slices.ContainsFunc(ret.Values, isErrExprAST) {
		sp := ast.SourceSpan{}
		for _, v := range ret.Values {
			if isErrExprAST(v) {
				sp = spanOfExpression(v)
				break
			}
		}
		if !sp.IsSet() && tc.currentFunction != nil {
			sp = tc.currentFunction.Ident.Span
		}
		return reportf(sp, "result-err-branch-return",
			"use ensure to propagate Result failures",
			"Inside an `if r is Err(...)` branch, do not `return Err(...)`. Propagate with ensure instead.",
			"write `ensure x is Ok()` (or `ensure … else err`) instead of `if` + `return Err(...)`")
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
