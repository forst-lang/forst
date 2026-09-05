// Defs lookups: optional type registration, type guards, structural identity by shape.
package typechecker

import (
	"forst/internal/ast"
	"strings"
)

// RegisterTypeIfMissing registers a type definition if not already present in Defs.
// Accepts either ast.TypeDefNode or ast.TypeDefShapeExpr as def.
func (tc *TypeChecker) RegisterTypeIfMissing(ident ast.TypeIdent, def any) {
	if _, exists := tc.Defs[ident]; exists {
		return
	}
	switch d := def.(type) {
	case ast.TypeDefNode:
		tc.setDef(ident, d)
	case ast.TypeDefShapeExpr:
		tc.setDef(ident, d)
	default:
		panic("RegisterTypeIfMissing: unsupported type definition")
	}
}

// IsTypeGuardConstraint returns true if the given constraint name is a registered type guard
func (tc *TypeChecker) IsTypeGuardConstraint(name string) bool {
	if def, exists := tc.Defs[ast.TypeIdent(name)]; exists {
		if _, ok := def.(ast.TypeGuardNode); ok {
			return true
		}
		if _, ok := def.(*ast.TypeGuardNode); ok {
			return true
		}
	}
	return false
}

// FindStructurallyIdenticalNamedType returns the first user-defined named type that is structurally identical to the given hash-based type, or "" if none.
func (tc *TypeChecker) FindStructurallyIdenticalNamedType(typeNode ast.TypeNode) ast.TypeIdent {
	if !typeNode.IsHashBased() {
		return ""
	}
	if alias, ok := tc.lookupShapeAliasForHashType(typeNode); ok {
		return alias
	}
	return ""
}

// FindAnyStructurallyIdenticalNamedType returns the first user-defined named type that is structurally identical to the given shape, or "" if none.
// This function works for any shape, not just hash-based types.
func (tc *TypeChecker) FindAnyStructurallyIdenticalNamedType(shape ast.ShapeNode) ast.TypeIdent {
	h, err := tc.Hasher.HashNode(shape)
	if err != nil {
		return ""
	}
	if alias, ok := tc.shapeAliasIndexOrBuild().byShapeHash[h]; ok {
		return alias
	}
	return ""
}

// UserNamedTypeMatchesShape reports whether ident names a user type whose shape is structurally identical to shape.
func (tc *TypeChecker) UserNamedTypeMatchesShape(ident ast.TypeIdent, shape ast.ShapeNode) bool {
	if ident == "" || strings.HasPrefix(string(ident), "T_") {
		return false
	}
	def, ok := tc.Defs[ident]
	if !ok {
		return false
	}
	typeDef, ok := def.(ast.TypeDefNode)
	if !ok {
		return false
	}
	payload, ok := ast.PayloadShape(typeDef.Expr)
	if !ok {
		return false
	}
	return tc.shapesAreStructurallyIdentical(shape, *payload)
}
