package typechecker

import (
	"fmt"

	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

// applyIfBranchNarrowing registers a shadow variable binding in the current scope when the
// condition is of the form `subject is <assertion|shape>`. Strategy: lexical shadowing in the
// branch scope (see ROADMAP / type narrowing plan) so LookupVariableType sees the refined type.
// The condition must already have been type-checked (unifyIsOperator validation).
func (tc *TypeChecker) applyIfBranchNarrowing(condition ast.Node) {
	if condition == nil {
		return
	}
	bin, ok := condition.(ast.BinaryExpressionNode)
	if !ok || bin.Operator != ast.TokenIs {
		return
	}
	refined, err := tc.refinedTypesForIsNarrowing(bin.Left, bin.Right)
	if err != nil {
		tc.log.WithFields(logrus.Fields{
			"function": "applyIfBranchNarrowing",
		}).WithError(err).Debug("skipping if-branch narrowing")
		return
	}
	if len(refined) == 0 {
		return
	}
	lv, err := tc.getLeftmostVariable(bin.Left)
	if err != nil {
		return
	}
	vn, ok := lv.(ast.VariableNode)
	if !ok {
		return
	}
	tc.scopeStack.currentScope().RegisterSymbol(vn.Ident.ID, refined, SymbolVariable)
}

// refinedTypesForIsNarrowing returns the type(s) the subject should have when the `is` condition
// is true. Reuses InferAssertionType / inferShapeType so narrowing stays aligned with assertions.
func (tc *TypeChecker) refinedTypesForIsNarrowing(left, right ast.Node) ([]ast.TypeNode, error) {
	leftmostVar, err := tc.getLeftmostVariable(left)
	if err != nil {
		return nil, err
	}
	varLeftTypes, err := tc.inferExpressionType(leftmostVar)
	if err != nil {
		return nil, err
	}
	if len(varLeftTypes) != 1 {
		return nil, fmt.Errorf("expected a single type for `is` subject, got %d", len(varLeftTypes))
	}
	varLeftType := varLeftTypes[0]

	switch r := right.(type) {
	case ast.TypeDefAssertionExpr:
		if r.Assertion == nil {
			return nil, fmt.Errorf("missing assertion on RHS of `is`")
		}
		return tc.InferAssertionType(r.Assertion, false, "", &varLeftType)
	case ast.AssertionNode:
		return tc.InferAssertionType(&r, false, "", &varLeftType)
	case ast.ShapeNode:
		tn, err := tc.inferShapeType(r, &varLeftType)
		if err != nil {
			return nil, err
		}
		return []ast.TypeNode{tn}, nil
	default:
		rightTypes, err := tc.inferExpressionType(right)
		if err != nil {
			return nil, err
		}
		if len(rightTypes) != 1 {
			return nil, fmt.Errorf("expected single type on RHS of `is`")
		}
		if rightTypes[0].Ident == ast.TypeShape {
			return nil, fmt.Errorf("shape RHS must be a ShapeNode for narrowing, got %T", right)
		}
		return nil, fmt.Errorf("unsupported RHS for narrowing: %T", right)
	}
}
