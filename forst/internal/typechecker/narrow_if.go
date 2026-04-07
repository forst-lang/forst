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
//
// When the assertion uses only user-defined type guard constraints (e.g. `x is MyGuard()`), we
// refine to the guard's subject parameter type (see refinedNarrowingWhenAssertionUsesOnlyTypeGuards)
// instead of InferAssertionType's merged structural hash, so builtins like println still see String.
func (tc *TypeChecker) refinedNarrowingTypeFromAliasAssertion(assertion *ast.AssertionNode, refined []ast.TypeNode) []ast.TypeNode {
	if assertion == nil {
		return refined
	}
	if len(assertion.Constraints) > 0 {
		if alt := tc.refinedNarrowingWhenAssertionUsesOnlyTypeGuards(assertion); len(alt) > 0 {
			return alt
		}
		return refined
	}
	if assertion.BaseType == nil {
		return refined
	}
	base := *assertion.BaseType
	// `if x is MyGuard` often parses as BaseType = guard name (no constraints). Type guards live in
	// Defs as TypeGuardNode, not TypeDefNode, so alias-chain resolution above does not apply.
	if tc.IsTypeGuardConstraint(string(base)) {
		if t := tc.refinedTypesFromTypeGuardName(ast.TypeIdent(base)); len(t) > 0 {
			return t
		}
	}
	if u := tc.underlyingBuiltinTypeOfAliasAssertion(base); u != "" {
		return []ast.TypeNode{{Ident: u}}
	}
	return refined
}

// refinedTypesFromTypeGuardName returns the subject parameter type for a registered type guard,
// resolving alias-of-builtin to the builtin (e.g. Password → String) for consistent narrowing.
func (tc *TypeChecker) refinedTypesFromTypeGuardName(guardName ast.TypeIdent) []ast.TypeNode {
	def, ok := tc.Defs[guardName]
	if !ok {
		return nil
	}
	var gn ast.TypeGuardNode
	switch d := def.(type) {
	case *ast.TypeGuardNode:
		gn = *d
	case ast.TypeGuardNode:
		gn = d
	default:
		return nil
	}
	sp, ok := gn.Subject.(ast.SimpleParamNode)
	if !ok {
		return nil
	}
	tn := sp.Type
	if u := tc.underlyingBuiltinTypeOfAliasAssertion(tn.Ident); u != "" {
		return []ast.TypeNode{{Ident: u}}
	}
	return []ast.TypeNode{tn}
}

// refinedNarrowingWhenAssertionUsesOnlyTypeGuards applies when every constraint is a user type guard;
// the narrowed static type is then the first guard's subject type (same as the declared subject).
func (tc *TypeChecker) refinedNarrowingWhenAssertionUsesOnlyTypeGuards(assertion *ast.AssertionNode) []ast.TypeNode {
	if assertion == nil || len(assertion.Constraints) == 0 {
		return nil
	}
	for _, c := range assertion.Constraints {
		if !tc.IsTypeGuardConstraint(c.Name) {
			return nil
		}
	}
	return tc.refinedTypesFromTypeGuardName(ast.TypeIdent(assertion.Constraints[0].Name))
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
	disp := tc.narrowingPredicateDisplayFromIsRHS(bin.Right)
	tc.scopeStack.currentScope().RegisterSymbolWithNarrowing(vn.Ident.ID, refined, SymbolVariable, guards, disp)
	tc.recordCompoundNarrowingIdentifier(vn.Ident.ID, guards, disp)
	tc.recordIfChainNarrowingSubject(vn.Ident.ID, refined, guards)
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
	case ast.AssertionNode:
		if handled, refined, err := tc.refinedTypesForResultIsNarrowing(varLeftType, &r); handled {
			if err != nil {
				return nil, err
			}
			return refined, nil
		}
		vn, ok := leftmostVar.(ast.VariableNode)
		if !ok {
			return nil, fmt.Errorf("assertion RHS narrowing requires variable subject, got %T", leftmostVar)
		}
		return tc.refinedTypesForAssertionOnVariable(vn, &r)
	case ast.TypeDefAssertionExpr:
		if r.Assertion == nil {
			return nil, fmt.Errorf("missing assertion on RHS of `is`")
		}
		refined, err := tc.InferAssertionType(r.Assertion, false, "", &varLeftType)
		if err != nil {
			return nil, err
		}
		return tc.refinedNarrowingTypeFromAliasAssertion(r.Assertion, refined), nil
	case ast.ShapeNode:
		tn, err := tc.inferShapeType(r, &varLeftType)
		if err != nil {
			return nil, err
		}
		return []ast.TypeNode{tn}, nil
	case ast.FunctionCallNode:
		if tc.IsTypeGuardConstraint(string(r.Function.ID)) {
			if t := tc.refinedTypesFromTypeGuardName(ast.TypeIdent(r.Function.ID)); len(t) > 0 {
				return t, nil
			}
		}
		return nil, fmt.Errorf("unsupported RHS for narrowing: %T", right)
	case *ast.FunctionCallNode:
		if r == nil {
			return nil, fmt.Errorf("nil RHS in `is` narrowing")
		}
		if tc.IsTypeGuardConstraint(string(r.Function.ID)) {
			if t := tc.refinedTypesFromTypeGuardName(ast.TypeIdent(r.Function.ID)); len(t) > 0 {
				return t, nil
			}
		}
		return nil, fmt.Errorf("unsupported RHS for narrowing: %T", right)
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

// refinedTypesForResultIsNarrowing handles `x is Ok(...)` / `Err(...)` when x is Result(S,F).
func (tc *TypeChecker) refinedTypesForResultIsNarrowing(varLeftType ast.TypeNode, a *ast.AssertionNode) (handled bool, refined []ast.TypeNode, err error) {
	if a == nil || len(a.Constraints) != 1 || a.BaseType != nil {
		return false, nil, nil
	}
	c := a.Constraints[0]
	if c.Name != "Ok" && c.Name != "Err" {
		return false, nil, nil
	}
	if !varLeftType.IsResultType() || len(varLeftType.TypeParams) < 2 {
		// User type guard named Ok/Err (e.g. `is (v N) Ok()`) — not Result discriminators.
		return false, nil, nil
	}
	if err := tc.validateResultDiscriminatorAssertion(*a, varLeftType); err != nil {
		return true, nil, err
	}
	if c.Name == "Ok" {
		return true, []ast.TypeNode{varLeftType.TypeParams[0]}, nil
	}
	return true, []ast.TypeNode{varLeftType.TypeParams[1]}, nil
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
	// Keep successor scope and hover on the subject's static type (e.g. String) for built-in
	// refinements (Min, Max, …). InferAssertionType still produces hash structural types for codegen.
	if tc.assertionRefinesBuiltinSubjectWithOnlyBuiltinConstraints(assertion) {
		return []ast.TypeNode{varLeftType}, nil
	}
	refined, err := tc.InferAssertionType(assertion, false, "", &varLeftType)
	if err != nil {
		return nil, err
	}
	return tc.refinedNarrowingTypeFromAliasAssertion(assertion, refined), nil
}

// assertionRefinesBuiltinSubjectWithOnlyBuiltinConstraints matches assertions like
// `String.Min(12)`, `ensure x is Min(1)` (no BaseType), or `ensure x is String.Min(1)` where every
// constraint is a built-in refinement, not a user TypeGuardNode.
func (tc *TypeChecker) assertionRefinesBuiltinSubjectWithOnlyBuiltinConstraints(a *ast.AssertionNode) bool {
	if a == nil || len(a.Constraints) == 0 {
		return false
	}
	for _, c := range a.Constraints {
		if c.Name == ConstraintMatch {
			return false
		}
		if def, ok := tc.Defs[ast.TypeIdent(c.Name)]; ok {
			if _, ok := def.(ast.TypeGuardNode); ok {
				return false
			}
		}
		if !isBuiltinAssertionConstraintName(c.Name) {
			return false
		}
	}
	if a.BaseType != nil && !tc.isBuiltinType(*a.BaseType) {
		return false
	}
	return true
}

// applyEnsureSuccessorNarrowing registers a refined binding for the ensure subject so that
// following statements (or the ensure block body) see the same types as after `x is …`.
// Field paths such as `g.cells` register under the full identifier so lookup + hover match
// simple variables (Min/Max chain, etc.).
func (tc *TypeChecker) applyEnsureSuccessorNarrowing(n ast.EnsureNode) {
	vn := n.Variable
	// Best-effort assertion expression inference (registers tc.Types for the assertion subtree).
	// Do not abort narrowing on failure: `inferExpressionType(AssertionNode)` often lacks the
	// subject context that `refinedTypesForIsNarrowing` supplies via InferAssertionType, so it can
	// error while validation + narrowing still succeed (e.g. `ensure x is String.Min(1)`).
	if _, err := tc.inferExpressionType(n.Assertion); err != nil {
		tc.log.WithFields(logrus.Fields{
			"function": "applyEnsureSuccessorNarrowing",
		}).WithError(err).Debug("ensure assertion expr inference failed (continuing with narrowing)")
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
	disp := tc.narrowingPredicateDisplayFromIsRHS(n.Assertion)
	tc.scopeStack.currentScope().RegisterSymbolWithNarrowing(vn.Ident.ID, refined, SymbolVariable, guards, disp)
	tc.recordCompoundNarrowingIdentifier(vn.Ident.ID, guards, disp)
}

func (tc *TypeChecker) recordCompoundNarrowingIdentifier(id ast.Identifier, guards []string, disp string) {
	if tc == nil || id == "" || !strings.Contains(string(id), ".") {
		return
	}
	prev := tc.compoundNarrowingByIdentifier[id]
	mergedGuards := mergeNarrowingGuardNamesDedupe(prev.guards, guards)
	mergedDisp := mergeNarrowingPredicateDisplaySegments(prev.disp, disp)
	tc.compoundNarrowingByIdentifier[id] = compoundNarrowingInfo{
		guards: mergedGuards,
		disp:   mergedDisp,
	}
}

// --- Control-flow join at the merge point after a completed if / else-if / else chain (plan §3.2).

// narrowingEvent records one successful `x is …` branch narrowing in the current if-chain.
type narrowingEvent struct {
	ident               ast.Identifier
	refined             []ast.TypeNode
	narrowingTypeGuards []string
}

func (tc *TypeChecker) beginIfChainForStatement() {
	tc.ifChainNarrowingStack = append(tc.ifChainNarrowingStack, nil)
}

func (tc *TypeChecker) recordIfChainNarrowingSubject(id ast.Identifier, refined []ast.TypeNode, guards []string) {
	if len(tc.ifChainNarrowingStack) == 0 || len(refined) == 0 {
		return
	}
	top := len(tc.ifChainNarrowingStack) - 1
	tc.ifChainNarrowingStack[top] = append(tc.ifChainNarrowingStack[top], narrowingEvent{
		ident:               id,
		refined:             append([]ast.TypeNode(nil), refined...),
		narrowingTypeGuards: append([]string(nil), guards...),
	})
}

// endIfChainApplyJoin runs at the syntactic merge point after an IfNode: for each binding that was
// narrowed in any branch, the static type after the whole chain is JoinAfterIfMerge → outer
// (pre-if) type. LookupVariable already resolves to the outer binding; we also run
// MergeFlowFactsAtIfJoin with actual branch refinements so JoinAfterIfMerge receives real inputs
// (today still widens to outer; future union/LUB can use branchRefinements).
func (tc *TypeChecker) endIfChainApplyJoin() {
	if len(tc.ifChainNarrowingStack) == 0 {
		return
	}
	top := len(tc.ifChainNarrowingStack) - 1
	frame := tc.ifChainNarrowingStack[top]
	tc.ifChainNarrowingStack = tc.ifChainNarrowingStack[:top]
	if len(frame) == 0 {
		return
	}

	seenID := make(map[ast.Identifier]struct{}, len(frame))
	var branchFacts []FlowTypeFact
	for _, ev := range frame {
		if _, ok := seenID[ev.ident]; !ok {
			seenID[ev.ident] = struct{}{}
		}
		for _, rt := range ev.refined {
			branchFacts = append(branchFacts, FlowTypeFact{
				Ident:               ev.ident,
				RefinedType:         rt,
				NarrowingTypeGuards: append([]string(nil), ev.narrowingTypeGuards...),
			})
		}
	}

	refinementCount := make(map[ast.Identifier]int, len(seenID))
	for _, f := range branchFacts {
		refinementCount[f.Ident]++
	}

	outerByIdent := make(map[ast.Identifier]ast.TypeNode, len(seenID))
	for id := range seenID {
		sym, ok := tc.CurrentScope().LookupVariable(id)
		if !ok || len(sym.Types) != 1 {
			continue
		}
		outerByIdent[id] = sym.Types[0]
	}

	mergedByIdent := MergeFlowFactsAtIfJoin(outerByIdent, branchFacts)
	for id, merged := range mergedByIdent {
		outer := outerByIdent[id]
		tc.log.WithFields(logrus.Fields{
			"function":          "endIfChainApplyJoin",
			"identifier":        id,
			"outer":             outer.Ident,
			"merged":            merged.Ident,
			"branchRefinements": refinementCount[id],
		}).Trace("if-chain merge point: MergeFlowFactsAtIfJoin + JoinAfterIfMerge (widen to outer / pre-if binding)")
	}
}
