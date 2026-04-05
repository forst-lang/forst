package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

// underlyingBuiltinTypeOfAliasAssertion returns the built-in type ident (e.g. String) when a named
// type is defined only as a TypeDefAssertionExpr over a built-in (or a chain of such aliases)
// with no constraints on that assertion. Otherwise returns empty.
func (tc *TypeChecker) underlyingBuiltinTypeOfAliasAssertion(alias ast.TypeIdent) ast.TypeIdent {
	def, ok := tc.Defs[alias].(ast.TypeDefNode)
	if !ok {
		return ""
	}
	var ade *ast.TypeDefAssertionExpr
	switch expr := def.Expr.(type) {
	case ast.TypeDefAssertionExpr:
		e := expr
		ade = &e
	case *ast.TypeDefAssertionExpr:
		ade = expr
	default:
		return ""
	}
	if ade == nil || ade.Assertion == nil {
		return ""
	}
	ann := ade.Assertion
	if ann.BaseType == nil || len(ann.Constraints) != 0 {
		return ""
	}
	if tc.isBuiltinType(*ann.BaseType) {
		return *ann.BaseType
	}
	return tc.underlyingBuiltinTypeOfAliasAssertion(ast.TypeIdent(*ann.BaseType))
}

// refinedNarrowingTypeFromAliasAssertion maps `x is MyStr`-style narrowing to the underlying
// built-in when MyStr is an alias defined as an assertion over that built-in. This matches
// refinement semantics (underlying type in the branch) and avoids registering a hash-only T_…
// type as the narrowed binding when InferAssertionType would otherwise fall back to a structural hash.
func (tc *TypeChecker) refinedNarrowingTypeFromAliasAssertion(assertion *ast.AssertionNode, refined []ast.TypeNode) []ast.TypeNode {
	if assertion == nil || len(assertion.Constraints) != 0 {
		return refined
	}
	if assertion.BaseType == nil {
		return refined
	}
	if u := tc.underlyingBuiltinTypeOfAliasAssertion(*assertion.BaseType); u != "" {
		return []ast.TypeNode{{Ident: u}}
	}
	return refined
}

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
	guards := tc.typeGuardNamesFromIsRHS(bin.Right)
	tc.scopeStack.currentScope().RegisterSymbolWithNarrowing(vn.Ident.ID, refined, SymbolVariable, guards)
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
		refined, err := tc.InferAssertionType(r.Assertion, false, "", &varLeftType)
		if err != nil {
			return nil, err
		}
		return tc.refinedNarrowingTypeFromAliasAssertion(r.Assertion, refined), nil
	case ast.AssertionNode:
		vn, ok := leftmostVar.(ast.VariableNode)
		if !ok {
			return nil, fmt.Errorf("assertion RHS narrowing requires variable subject, got %T", leftmostVar)
		}
		return tc.refinedTypesForAssertionOnVariable(vn, &r)
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

// refinedTypesForAssertionOnVariable returns the refined type(s) for a variable under the given
// assertion (same rules as the RHS of `x is …`). Shared by if-branch narrowing and ensure-successor narrowing.
func (tc *TypeChecker) refinedTypesForAssertionOnVariable(vn ast.VariableNode, assertion *ast.AssertionNode) ([]ast.TypeNode, error) {
	if assertion == nil {
		return nil, fmt.Errorf("nil assertion")
	}
	varLeftTypes, err := tc.inferExpressionType(vn)
	if err != nil {
		return nil, err
	}
	if len(varLeftTypes) != 1 {
		return nil, fmt.Errorf("expected a single type for assertion subject, got %d", len(varLeftTypes))
	}
	varLeftType := varLeftTypes[0]
	refined, err := tc.InferAssertionType(assertion, false, "", &varLeftType)
	if err != nil {
		return nil, err
	}
	return tc.refinedNarrowingTypeFromAliasAssertion(assertion, refined), nil
}

// applyEnsureSuccessorNarrowing registers a refined binding for the ensure subject so that
// following statements (or the ensure block body) see the same types as after `x is …`.
// Compound subjects (field paths) are skipped until we have a dedicated occurrence model for them.
func (tc *TypeChecker) applyEnsureSuccessorNarrowing(n ast.EnsureNode) {
	vn := n.Variable
	if strings.Contains(string(vn.Ident.ID), ".") {
		tc.log.WithFields(logrus.Fields{
			"function": "applyEnsureSuccessorNarrowing",
			"subject":  vn.Ident.ID,
		}).Debug("skipping ensure successor narrowing for compound subject")
		return
	}
	// Match `if x is <assertion>`: infer the assertion as an expression first (see inferIfStatement
	// ordering), then reuse the same narrowing as the `is` branch.
	if _, err := tc.inferExpressionType(n.Assertion); err != nil {
		tc.log.WithFields(logrus.Fields{
			"function": "applyEnsureSuccessorNarrowing",
		}).WithError(err).Debug("skipping ensure successor narrowing (assertion expression)")
		return
	}
	refined, err := tc.refinedTypesForIsNarrowing(n.Variable, n.Assertion)
	if err != nil {
		tc.log.WithFields(logrus.Fields{
			"function": "applyEnsureSuccessorNarrowing",
		}).WithError(err).Debug("skipping ensure successor narrowing")
		return
	}
	if len(refined) == 0 {
		return
	}
	guards := tc.typeGuardNamesFromAssertionNode(&n.Assertion)
	tc.scopeStack.currentScope().RegisterSymbolWithNarrowing(vn.Ident.ID, refined, SymbolVariable, guards)
}
