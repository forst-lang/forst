package typechecker

import (
	"fmt"

	"forst/internal/ast"
)

// isEnsureScalarCarrierIdent reports whether ident is a builtin scalar that may host
// analyzable Min/Max-style constraints as named ensure type targets (not Shape/Array/Map/…).
func isEnsureScalarCarrierIdent(ident ast.TypeIdent) bool {
	switch ident {
	case ast.TypeString, ast.TypeInt, ast.TypeFloat, ast.TypeBool:
		return true
	default:
		return false
	}
}

// isEnsureScalarCarrier reports whether base is String/Int/Float/Bool, or an alias of one.
func (tc *TypeChecker) isEnsureScalarCarrier(base ast.TypeIdent) bool {
	if isEnsureScalarCarrierIdent(base) {
		return true
	}
	if u := tc.underlyingBuiltinTypeOfAliasAssertion(base); u != "" {
		return isEnsureScalarCarrierIdent(u)
	}
	return false
}

// carrierTypeForNamedType returns the runtime representation type for a named domain
// (literal union → String/Int/Bool; type Password = String → String;
// type Sku = String.Min(1).Max(64) → String).
func (tc *TypeChecker) carrierTypeForNamedType(t ast.TypeNode) (ast.TypeNode, bool) {
	if members, ok := tc.literalUnionMembers(t); ok && len(members) > 0 {
		switch literalKindOfValue(members[0]) {
		case literalKindString:
			return ast.TypeNode{Ident: ast.TypeString, TypeKind: ast.TypeKindBuiltin}, true
		case literalKindInt:
			return ast.TypeNode{Ident: ast.TypeInt, TypeKind: ast.TypeKindBuiltin}, true
		case literalKindBool:
			return ast.TypeNode{Ident: ast.TypeBool, TypeKind: ast.TypeKindBuiltin}, true
		}
	}
	def, ok := tc.Defs[t.Ident].(ast.TypeDefNode)
	if !ok {
		return ast.TypeNode{}, false
	}
	ade := typeDefAssertionExprOf(def)
	if ade == nil || ade.Assertion == nil || ade.Assertion.BaseType == nil {
		return ast.TypeNode{}, false
	}
	base := *ade.Assertion.BaseType
	if len(ade.Assertion.Constraints) > 0 {
		// Constrained aliases only expose a carrier when the base is a scalar domain.
		if !tc.isEnsureScalarCarrier(base) {
			return ast.TypeNode{}, false
		}
		if tc.isBuiltinType(base) {
			return ast.TypeNode{Ident: base, TypeKind: ast.TypeKindBuiltin}, true
		}
		if u := tc.underlyingBuiltinTypeOfAliasAssertion(base); u != "" {
			return ast.TypeNode{Ident: u, TypeKind: ast.TypeKindBuiltin}, true
		}
		return ast.TypeNode{}, false
	}
	if tc.isBuiltinType(base) {
		return ast.TypeNode{Ident: base, TypeKind: ast.TypeKindBuiltin}, true
	}
	return tc.carrierTypeForNamedType(ast.TypeNode{Ident: base, TypeKind: ast.TypeKindUserDefined})
}

// typeDefAssertionExprOf returns the assertion expr on a typedef, if any.
func typeDefAssertionExprOf(def ast.TypeDefNode) *ast.TypeDefAssertionExpr {
	switch expr := def.Expr.(type) {
	case ast.TypeDefAssertionExpr:
		e := expr
		return &e
	case *ast.TypeDefAssertionExpr:
		return expr
	default:
		return nil
	}
}

// ConstrainedScalarAliasAssertion returns the Meet assertion body for a named
// refined scalar alias such as `type Sku = String.Min(1).Max(64)`.
// The typedef must have a non-empty Meet constraint chain (no Join) over a
// scalar builtin carrier (directly or via alias). Inherited Meet constraints
// from scalar-alias bases are merged before lowering; recursive aliases fail closed.
func (tc *TypeChecker) ConstrainedScalarAliasAssertion(name ast.TypeIdent) (*ast.AssertionNode, bool) {
	return tc.constrainedScalarAliasAssertion(name)
}

func (tc *TypeChecker) constrainedScalarAliasAssertion(name ast.TypeIdent) (*ast.AssertionNode, bool) {
	return tc.constrainedScalarAliasAssertionSeen(name, map[ast.TypeIdent]struct{}{})
}

func (tc *TypeChecker) constrainedScalarAliasAssertionSeen(name ast.TypeIdent, seen map[ast.TypeIdent]struct{}) (*ast.AssertionNode, bool) {
	if _, dup := seen[name]; dup {
		return nil, false
	}
	seen[name] = struct{}{}

	def, ok := tc.Defs[name].(ast.TypeDefNode)
	if !ok {
		return nil, false
	}
	ade := typeDefAssertionExprOf(def)
	if ade == nil || ade.Assertion == nil || ade.Assertion.BaseType == nil {
		return nil, false
	}
	if len(ade.Assertion.OrChains) > 0 {
		return nil, false
	}
	base := *ade.Assertion.BaseType
	own := ade.Assertion.Constraints

	var inherited []ast.ConstraintNode
	var carrier ast.TypeIdent
	if tc.isBuiltinType(base) {
		carrier = base
	} else if _, cycling := seen[base]; cycling {
		return nil, false
	} else if inh, ok := tc.constrainedScalarAliasAssertionSeen(base, seen); ok && inh != nil && inh.BaseType != nil {
		inherited = inh.Constraints
		carrier = *inh.BaseType
	} else if tc.isEnsureScalarCarrier(base) {
		if u := tc.underlyingBuiltinTypeOfAliasAssertion(base); u != "" {
			carrier = u
		}
	}

	merged := append(append([]ast.ConstraintNode{}, inherited...), own...)
	if len(merged) == 0 || !isEnsureScalarCarrierIdent(carrier) {
		return nil, false
	}
	return &ast.AssertionNode{BaseType: &carrier, Constraints: merged}, true
}

// isBareNominalScalarDomain reports type Password = String (builtin base, no constraints).
func (tc *TypeChecker) isBareNominalScalarDomain(name ast.TypeIdent) bool {
	def, ok := tc.Defs[name].(ast.TypeDefNode)
	if !ok {
		return false
	}
	ade := typeDefAssertionExprOf(def)
	if ade == nil || ade.Assertion == nil || ade.Assertion.BaseType == nil {
		return false
	}
	return len(ade.Assertion.Constraints) == 0 &&
		len(ade.Assertion.OrChains) == 0 &&
		tc.isBuiltinType(*ade.Assertion.BaseType)
}

// isRuntimeEnsureTypeTarget reports whether named type may appear as `ensure x is T`
// (literal unions, enum subsets, nominal scalar domains, constrained scalar aliases —
// not arbitrary shapes).
func (tc *TypeChecker) isRuntimeEnsureTypeTarget(name ast.TypeIdent) bool {
	t := ast.TypeNode{Ident: name, TypeKind: ast.TypeKindUserDefined}
	if tc.isLiteralUnionType(t) {
		return true
	}
	if _, ok := tc.constrainedScalarAliasAssertion(name); ok {
		return true
	}
	if tc.isBareNominalScalarDomain(name) {
		return true
	}
	return false
}

// validateEnsureTypeTarget checks RFC 14 type-target rules for `ensure x is T`.
func (tc *TypeChecker) validateEnsureTypeTarget(ensure ast.EnsureNode, subjectType ast.TypeNode) error {
	tt, ok := ensure.Target.(ast.TypeTarget)
	if !ok {
		if p, ok := ensure.Target.(*ast.TypeTarget); ok && p != nil {
			tt = *p
		} else {
			return nil
		}
	}
	name := tt.Name
	if def, ok := tc.Defs[name]; ok {
		if _, isGuard := def.(ast.TypeGuardNode); isGuard {
			return reportBodyf(ensure.Variable.Ident.Span, "refinement-bare-guard-needs-parens",
				"refinement-bare-guard-needs-parens: guard %s requires parentheses — use `%s()` for an assertion, or a type name for a type target",
				name, name)
		}
		if _, isGuard := def.(*ast.TypeGuardNode); isGuard {
			return reportBodyf(ensure.Variable.Ident.Span, "refinement-bare-guard-needs-parens",
				"refinement-bare-guard-needs-parens: guard %s requires parentheses — use `%s()` for an assertion, or a type name for a type target",
				name, name)
		}
	}
	if !tc.isRuntimeEnsureTypeTarget(name) {
		return reportBodyf(ensure.Variable.Ident.Span, "refinement-type-target-not-runtime",
			"refinement-type-target-not-runtime: type %s is not a runtime ensure target; use a literal-union, enum subset, nominal scalar domain, or constrained scalar alias",
			name)
	}
	target := ast.TypeNode{Ident: name, TypeKind: ast.TypeKindUserDefined}
	carrier, ok := tc.carrierTypeForNamedType(target)
	if !ok {
		return fmt.Errorf("type target %s has no runtime carrier", name)
	}
	// Subject must be representation-compatible with the carrier (String → TaskStatus).
	if !tc.IsTypeCompatible(subjectType, carrier) && !tc.IsTypeCompatible(subjectType, target) {
		// Also allow already-narrowed same-domain values.
		subjCarrier, subOk := tc.carrierTypeForNamedType(subjectType)
		if !subOk || subjCarrier.Ident != carrier.Ident {
			return fmt.Errorf("type target %s expects carrier %s, got %s", name, formatTypeIdentForDiag(carrier.Ident), formatTypeIdentForDiag(subjectType.Ident))
		}
	}
	return nil
}

// rejectTypeNameAsAssertionCall rejects `ensure x is ActiveStatus()` when ActiveStatus is a type.
func (tc *TypeChecker) rejectTypeNameAsAssertionCall(assertion ast.AssertionNode, subjectSpan ast.SourceSpan) error {
	// Join alternatives: legacy failure-`or` and other OrChains diagnostics own this site first.
	if len(assertion.OrChains) > 0 {
		return nil
	}
	if assertion.BaseType != nil || len(assertion.Constraints) != 1 {
		return nil
	}
	name := ast.TypeIdent(assertion.Constraints[0].Name)
	if _, isGuard := tc.Defs[name].(ast.TypeGuardNode); isGuard {
		return nil
	}
	if _, isGuard := tc.Defs[name].(*ast.TypeGuardNode); isGuard {
		return nil
	}
	if def, ok := tc.Defs[name].(ast.TypeDefNode); ok {
		// Built-in-like constraints are not TypeDefNodes.
		_ = def
		sp := subjectSpan
		if len(assertion.Constraints) > 0 {
			sp = firstSetSpan(constraintSpan(assertion.Constraints[0]), subjectSpan)
		}
		return reportBodyf(sp, "refinement-enum-variant-not-assertion",
			"refinement-enum-variant-not-assertion: %s is a type, not an assertion; use `ensure x is %s` (no parentheses)",
			name, name)
	}
	return nil
}
