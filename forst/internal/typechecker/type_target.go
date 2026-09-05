package typechecker

import (
	"fmt"

	"forst/internal/ast"
)

// carrierTypeForNamedType returns the runtime representation type for a named domain
// (literal union → String/Int/Bool; type Password = String → String).
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
	if ae, ok := def.Expr.(ast.TypeDefAssertionExpr); ok && ae.Assertion != nil &&
		ae.Assertion.BaseType != nil && len(ae.Assertion.Constraints) == 0 {
		base := *ae.Assertion.BaseType
		if tc.isBuiltinType(base) {
			return ast.TypeNode{Ident: base, TypeKind: ast.TypeKindBuiltin}, true
		}
		return tc.carrierTypeForNamedType(ast.TypeNode{Ident: base, TypeKind: ast.TypeKindUserDefined})
	}
	return ast.TypeNode{}, false
}

// isRuntimeEnsureTypeTarget reports whether named type may appear as `ensure x is T`
// (literal unions, enum subsets, and nominal scalar domains — not arbitrary shapes).
func (tc *TypeChecker) isRuntimeEnsureTypeTarget(name ast.TypeIdent) bool {
	t := ast.TypeNode{Ident: name, TypeKind: ast.TypeKindUserDefined}
	if tc.isLiteralUnionType(t) {
		return true
	}
	if _, ok := tc.carrierTypeForNamedType(t); ok {
		// Nominal scalar domain (Password = String) is a type target.
		def, ok := tc.Defs[name].(ast.TypeDefNode)
		if !ok {
			return false
		}
		ae, ok := def.Expr.(ast.TypeDefAssertionExpr)
		if !ok || ae.Assertion == nil {
			return false
		}
		return ae.Assertion.BaseType != nil && len(ae.Assertion.Constraints) == 0 &&
			tc.isBuiltinType(*ae.Assertion.BaseType)
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
			return diagnosticf(ensure.Variable.Ident.Span, "refinement-bare-guard-needs-parens",
				"refinement-bare-guard-needs-parens: guard %s requires parentheses — use `%s()` for an assertion, or a type name for a type target",
				name, name)
		}
		if _, isGuard := def.(*ast.TypeGuardNode); isGuard {
			return diagnosticf(ensure.Variable.Ident.Span, "refinement-bare-guard-needs-parens",
				"refinement-bare-guard-needs-parens: guard %s requires parentheses — use `%s()` for an assertion, or a type name for a type target",
				name, name)
		}
	}
	if !tc.isRuntimeEnsureTypeTarget(name) {
		return diagnosticf(ensure.Variable.Ident.Span, "refinement-type-target-not-runtime",
			"refinement-type-target-not-runtime: type %s is not a runtime ensure target; use a literal-union, enum subset, or nominal scalar domain",
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
			return fmt.Errorf("type target %s expects carrier %s, got %s", name, carrier.Ident, subjectType.Ident)
		}
	}
	return nil
}

// rejectTypeNameAsAssertionCall rejects `ensure x is ActiveStatus()` when ActiveStatus is a type.
func (tc *TypeChecker) rejectTypeNameAsAssertionCall(assertion ast.AssertionNode) error {
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
		return diagnosticf(ast.SourceSpan{}, "refinement-enum-variant-not-assertion",
			"refinement-enum-variant-not-assertion: %s is a type, not an assertion; use `ensure x is %s` (no parentheses)",
			name, name)
	}
	return nil
}
