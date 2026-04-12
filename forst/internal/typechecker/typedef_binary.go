package typechecker

import (
	"fmt"
	"forst/internal/ast"
)

// TypeDefExprToTypeNode lowers a type-definition expression (including `|` / `&`) to a TypeNode.
// Used for typedef bodies and for compatibility checks on Result failure parameters.
func (tc *TypeChecker) TypeDefExprToTypeNode(expr ast.TypeDefExpr) (ast.TypeNode, error) {
	if expr == nil {
		return ast.TypeNode{}, fmt.Errorf("nil type definition expression")
	}
	switch e := expr.(type) {
	case ast.TypeDefAssertionExpr:
		return tc.typeDefAssertionExprToTypeNode(e)
	case ast.TypeDefShapeExpr:
		hash, err := tc.Hasher.HashNode(e.Shape)
		if err != nil {
			return ast.TypeNode{}, fmt.Errorf("hash shape: %w", err)
		}
		return ast.TypeNode{Ident: hash.ToTypeIdent(), TypeKind: ast.TypeKindHashBased}, nil
	case ast.TypeDefErrorExpr:
		return ast.TypeNode{}, fmt.Errorf("anonymous error payload in type expression is not valid; use a named error type")
	case ast.TypeDefBinaryExpr:
		left, err := tc.TypeDefExprToTypeNode(e.Left)
		if err != nil {
			return ast.TypeNode{}, err
		}
		right, err := tc.TypeDefExprToTypeNode(e.Right)
		if err != nil {
			return ast.TypeNode{}, err
		}
		if e.IsDisjunction() {
			parts := append(flattenUnionForCombine(left), flattenUnionForCombine(right)...)
			return ast.NewUnionType(parts...), nil
		}
		return tc.meetTypeDefPair(left, right)
	default:
		return ast.TypeNode{}, fmt.Errorf("unsupported type definition expression %T", expr)
	}
}

func flattenUnionForCombine(t ast.TypeNode) []ast.TypeNode {
	if t.Ident == ast.TypeUnion && len(t.TypeParams) > 0 {
		var out []ast.TypeNode
		for _, m := range t.TypeParams {
			out = append(out, flattenUnionForCombine(m)...)
		}
		return out
	}
	return []ast.TypeNode{t}
}

func (tc *TypeChecker) typeDefAssertionExprToTypeNode(e ast.TypeDefAssertionExpr) (ast.TypeNode, error) {
	if e.Assertion == nil {
		return ast.TypeNode{}, fmt.Errorf("missing assertion in type definition")
	}
	if e.Assertion.BaseType != nil && len(e.Assertion.Constraints) == 0 {
		return ast.TypeNode{Ident: *e.Assertion.BaseType, TypeKind: ast.TypeKindUserDefined}, nil
	}
	return ast.NewAssertionType(e.Assertion), nil
}

func (tc *TypeChecker) meetTypeDefPair(a, b ast.TypeNode) (ast.TypeNode, error) {
	if m, ok := MeetTypes(a, b); ok {
		return m, nil
	}
	if m, ok := tc.MeetTypesSubtyping(a, b); ok {
		return m, nil
	}
	return ast.TypeNode{}, fmt.Errorf("intersection %s & %s is empty (unrelated types)", a.String(), b.String())
}

// MeetTypesSubtyping returns the greatest lower bound of a and b when one is a subtype of the other
// (e.g. nominal error <: Error), or when they are mutually assignable as equivalent.
func (tc *TypeChecker) MeetTypesSubtyping(a, b ast.TypeNode) (ast.TypeNode, bool) {
	if typeNodesShallowEqual(a, b) {
		return a, true
	}
	aSubB := tc.IsTypeCompatible(a, b)
	bSubA := tc.IsTypeCompatible(b, a)
	if aSubB && !bSubA {
		return a, true
	}
	if bSubA && !aSubB {
		return b, true
	}
	if aSubB && bSubA {
		return a, true
	}
	return ast.TypeNode{}, false
}

// IsErrorKindedType reports whether t is the built-in Error, a nominal error, or a union/intersection
// whose members are all error-kinded (for Result failure checking).
func (tc *TypeChecker) IsErrorKindedType(t ast.TypeNode) bool {
	if c, ok := tc.expandTypeDefBinaryIfNeeded(t); ok {
		t = c
	}
	if t.Ident == ast.TypeError {
		return true
	}
	if t.Ident == ast.TypeUnion {
		for _, m := range t.TypeParams {
			if !tc.IsErrorKindedType(m) {
				return false
			}
		}
		return len(t.TypeParams) > 0
	}
	if t.Ident == ast.TypeIntersection {
		for _, m := range t.TypeParams {
			if !tc.IsErrorKindedType(m) {
				return false
			}
		}
		return len(t.TypeParams) > 0
	}
	if def, ok := tc.Defs[t.Ident].(ast.TypeDefNode); ok {
		if _, ok := def.Expr.(ast.TypeDefErrorExpr); ok {
			return true
		}
	}
	return false
}

func (tc *TypeChecker) validateTypeDefBinary(ident ast.TypeIdent, expr ast.TypeDefBinaryExpr) error {
	_, err := tc.TypeDefExprToTypeNode(expr)
	if err != nil {
		return fmt.Errorf("type %s: %w", ident, err)
	}
	return nil
}

// expandTypeDefBinaryIfNeeded returns the canonical TypeNode for a typedef whose body is A | B or A & B.
func (tc *TypeChecker) expandTypeDefBinaryIfNeeded(t ast.TypeNode) (ast.TypeNode, bool) {
	if t.Ident == "" {
		return ast.TypeNode{}, false
	}
	def, ok := tc.Defs[t.Ident].(ast.TypeDefNode)
	if !ok {
		return ast.TypeNode{}, false
	}
	bin, ok := def.Expr.(ast.TypeDefBinaryExpr)
	if !ok {
		return ast.TypeNode{}, false
	}
	canon, err := tc.TypeDefExprToTypeNode(bin)
	if err != nil {
		return ast.TypeNode{}, false
	}
	// Do not expand self-referential typedefs (e.g. invalid U = U | T): canonical IR still mentions
	// this ident and would make IsTypeCompatible recurse forever (expand → Union → … → expand(U)).
	if containsTypeIdentParam(canon, t.Ident) {
		return ast.TypeNode{}, false
	}
	return canon, true
}

func containsTypeIdentParam(t ast.TypeNode, id ast.TypeIdent) bool {
	if t.Ident == id {
		return true
	}
	for i := range t.TypeParams {
		if containsTypeIdentParam(t.TypeParams[i], id) {
			return true
		}
	}
	return false
}
