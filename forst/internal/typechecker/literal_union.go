package typechecker

import (
	"fmt"

	"forst/internal/ast"
)

// literalKind classifies a typedef / type-node member for homogeneous literal unions.
type literalKind int

const (
	literalKindNone literalKind = iota
	literalKindString
	literalKindInt
	literalKindBool
	literalKindOther // named type, shape, assertion, etc.
)

// literalValueFromAssertion extracts a bare Value(literal) payload.
func literalValueFromAssertion(a *ast.AssertionNode) (ast.ValueNode, bool) {
	if a == nil || a.BaseType != nil || len(a.OrChains) != 0 {
		return nil, false
	}
	if len(a.Constraints) != 1 || a.Constraints[0].Name != ast.ValueConstraint {
		return nil, false
	}
	args := a.Constraints[0].Args
	if len(args) != 1 || args[0].Value == nil {
		return nil, false
	}
	v := *args[0].Value
	switch v.(type) {
	case ast.StringLiteralNode, *ast.StringLiteralNode,
		ast.IntLiteralNode, *ast.IntLiteralNode,
		ast.BoolLiteralNode, *ast.BoolLiteralNode:
		return v, true
	default:
		return nil, false
	}
}

// literalKindOfValue classifies a value node as string/int/bool/other.
func literalKindOfValue(v ast.ValueNode) literalKind {
	switch v.(type) {
	case ast.StringLiteralNode, *ast.StringLiteralNode:
		return literalKindString
	case ast.IntLiteralNode, *ast.IntLiteralNode:
		return literalKindInt
	case ast.BoolLiteralNode, *ast.BoolLiteralNode:
		return literalKindBool
	default:
		return literalKindNone
	}
}

// literalKindOfTypeNode classifies a type node for homogeneous union checks.
func literalKindOfTypeNode(t ast.TypeNode) literalKind {
	if t.Ident == ast.TypeAssertion && t.Assertion != nil {
		if v, ok := literalValueFromAssertion(t.Assertion); ok {
			return literalKindOfValue(v)
		}
	}
	if t.Ident == ast.TypeUnion {
		var seen literalKind
		for _, m := range t.TypeParams {
			k := literalKindOfTypeNode(m)
			if k == literalKindNone {
				return literalKindOther
			}
			if seen == literalKindNone {
				seen = k
				continue
			}
			if seen != k {
				return literalKindOther
			}
		}
		if seen == literalKindNone {
			return literalKindOther
		}
		return seen
	}
	return literalKindOther
}

// literalValuesEqual compares two literal value nodes structurally.
func literalValuesEqual(a, b ast.ValueNode) bool {
	switch av := a.(type) {
	case ast.StringLiteralNode:
		bv, ok := b.(ast.StringLiteralNode)
		if !ok {
			if p, ok := b.(*ast.StringLiteralNode); ok {
				bv = *p
			} else {
				return false
			}
		}
		return av.Value == bv.Value
	case *ast.StringLiteralNode:
		return literalValuesEqual(*av, b)
	case ast.IntLiteralNode:
		bv, ok := b.(ast.IntLiteralNode)
		if !ok {
			if p, ok := b.(*ast.IntLiteralNode); ok {
				bv = *p
			} else {
				return false
			}
		}
		return av.Value == bv.Value
	case *ast.IntLiteralNode:
		return literalValuesEqual(*av, b)
	case ast.BoolLiteralNode:
		bv, ok := b.(ast.BoolLiteralNode)
		if !ok {
			if p, ok := b.(*ast.BoolLiteralNode); ok {
				bv = *p
			} else {
				return false
			}
		}
		return av.Value == bv.Value
	case *ast.BoolLiteralNode:
		return literalValuesEqual(*av, b)
	default:
		return false
	}
}

// assertionTypesCompatible reports whether two assertion types can unify for membership.
func assertionTypesCompatible(a, b *ast.AssertionNode) bool {
	if a == nil || b == nil {
		return a == b
	}
	if av, ok := literalValueFromAssertion(a); ok {
		bv, ok2 := literalValueFromAssertion(b)
		return ok2 && literalValuesEqual(av, bv)
	}
	return a.String() == b.String()
}

// literalUnionMembers returns the Value literals of a named homogeneous literal union
// (or a TypeUnion of such members). ok is false when t is not a literal-union domain.
func (tc *TypeChecker) literalUnionMembers(t ast.TypeNode) ([]ast.ValueNode, bool) {
	if c, ok := tc.expandTypeDefBinaryIfNeeded(t); ok {
		t = c
	} else if def, ok := tc.Defs[t.Ident].(ast.TypeDefNode); ok {
		// Single-literal typedef: type Failed = "failed"
		if ae, ok := def.Expr.(ast.TypeDefAssertionExpr); ok && ae.Assertion != nil {
			if v, ok := literalValueFromAssertion(ae.Assertion); ok {
				return []ast.ValueNode{v}, true
			}
		}
	}
	if t.Ident == ast.TypeUnion {
		var out []ast.ValueNode
		var kind literalKind
		for _, m := range t.TypeParams {
			v, ok := literalValueFromAssertion(m.Assertion)
			if !ok || m.Ident != ast.TypeAssertion {
				return nil, false
			}
			k := literalKindOfValue(v)
			if kind == literalKindNone {
				kind = k
			} else if kind != k {
				return nil, false
			}
			out = append(out, v)
		}
		return out, len(out) > 0
	}
	if t.Ident == ast.TypeAssertion {
		if v, ok := literalValueFromAssertion(t.Assertion); ok {
			return []ast.ValueNode{v}, true
		}
	}
	return nil, false
}

// LiteralValueFromAssertion extracts a bare Value(literal) payload from an assertion.
func LiteralValueFromAssertion(a *ast.AssertionNode) (ast.ValueNode, bool) {
	return literalValueFromAssertion(a)
}

// LiteralUnionMembers returns the Value literals of a named homogeneous literal union.
func (tc *TypeChecker) LiteralUnionMembers(t ast.TypeNode) ([]ast.ValueNode, bool) {
	return tc.literalUnionMembers(t)
}

// isLiteralUnionType reports whether t names (or is) a homogeneous literal union.
func (tc *TypeChecker) isLiteralUnionType(t ast.TypeNode) bool {
	_, ok := tc.literalUnionMembers(t)
	return ok
}

// expressionLiteralValue returns the literal if expr is a string/int/bool literal.
func expressionLiteralValue(expr ast.Node) (ast.ValueNode, bool) {
	switch e := expr.(type) {
	case ast.StringLiteralNode, ast.IntLiteralNode, ast.BoolLiteralNode:
		return e.(ast.ValueNode), true
	case *ast.StringLiteralNode:
		return *e, true
	case *ast.IntLiteralNode:
		return *e, true
	case *ast.BoolLiteralNode:
		return *e, true
	default:
		return nil, false
	}
}

// literalAssignableToType reports whether a source literal is a member of expected's domain.
func (tc *TypeChecker) literalAssignableToType(lit ast.ValueNode, expected ast.TypeNode) bool {
	members, ok := tc.literalUnionMembers(expected)
	if !ok {
		return false
	}
	for _, m := range members {
		if literalValuesEqual(lit, m) {
			return true
		}
	}
	return false
}

// validateLiteralUnionExpr rejects deferred / non-homogeneous literal unions.
func (tc *TypeChecker) validateLiteralUnionExpr(ident ast.TypeIdent, expr ast.TypeDefExpr) error {
	bin, ok := expr.(ast.TypeDefBinaryExpr)
	if !ok {
		// Single literal member typedef is fine.
		if ae, ok := expr.(ast.TypeDefAssertionExpr); ok && ae.Assertion != nil {
			if _, ok := literalValueFromAssertion(ae.Assertion); ok {
				return nil
			}
		}
		return nil
	}
	if !bin.IsDisjunction() {
		// Intersection of literals is deferred.
		if containsLiteralTypeDefMember(bin) {
			return fmt.Errorf("type %s: refinement-unsupported-union: literal intersections are not supported", ident)
		}
		return nil
	}
	members := flattenTypeDefDisjuncts(bin)
	var litKind literalKind
	hasLit := false
	hasNonLit := false
	for _, m := range members {
		k := literalKindOfTypeDefExpr(m)
		switch k {
		case literalKindString, literalKindInt, literalKindBool:
			hasLit = true
			if litKind == literalKindNone {
				litKind = k
			} else if litKind != k {
				return fmt.Errorf("type %s: refinement-unsupported-union: mixed literal kinds in union", ident)
			}
		case literalKindOther:
			hasNonLit = true
		default:
			hasNonLit = true
		}
	}
	if hasLit && hasNonLit {
		return fmt.Errorf("type %s: refinement-unsupported-union: cannot mix literals with other type forms", ident)
	}
	return nil
}

// literalKindOfTypeDefExpr classifies a typedef expression member for union homogeneity.
func literalKindOfTypeDefExpr(e ast.TypeDefExpr) literalKind {
	switch x := e.(type) {
	case ast.TypeDefAssertionExpr:
		if x.Assertion == nil {
			return literalKindOther
		}
		if v, ok := literalValueFromAssertion(x.Assertion); ok {
			return literalKindOfValue(v)
		}
		return literalKindOther
	case *ast.TypeDefAssertionExpr:
		if x == nil {
			return literalKindOther
		}
		return literalKindOfTypeDefExpr(*x)
	case ast.TypeDefShapeExpr, ast.TypeDefErrorExpr:
		return literalKindOther
	case ast.TypeDefBinaryExpr:
		return literalKindOther
	default:
		return literalKindOther
	}
}

// containsLiteralTypeDefMember reports whether expr contains any literal union member.
func containsLiteralTypeDefMember(e ast.TypeDefExpr) bool {
	switch x := e.(type) {
	case ast.TypeDefBinaryExpr:
		return containsLiteralTypeDefMember(x.Left) || containsLiteralTypeDefMember(x.Right)
	default:
		k := literalKindOfTypeDefExpr(e)
		return k == literalKindString || k == literalKindInt || k == literalKindBool
	}
}

// flattenTypeDefDisjuncts collects | alternatives without changing declaration order.
func flattenTypeDefDisjuncts(e ast.TypeDefExpr) []ast.TypeDefExpr {
	bin, ok := e.(ast.TypeDefBinaryExpr)
	if !ok || !bin.IsDisjunction() {
		return []ast.TypeDefExpr{e}
	}
	left := flattenTypeDefDisjuncts(bin.Left)
	right := flattenTypeDefDisjuncts(bin.Right)
	out := make([]ast.TypeDefExpr, 0, len(left)+len(right))
	out = append(out, left...)
	out = append(out, right...)
	return out
}
