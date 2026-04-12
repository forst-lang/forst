// Defs lookups: optional type registration, type guards, structural identity by shape.
package typechecker

import (
	"forst/internal/ast"
	"strings"
)

// RegisterTypeIfMissing registers a type definition if not already present in Defs.
// Accepts either ast.TypeDefNode or ast.TypeDefShapeExpr as def.
func (tc *TypeChecker) RegisterTypeIfMissing(ident ast.TypeIdent, def interface{}) {
	if _, exists := tc.Defs[ident]; exists {
		return
	}
	switch d := def.(type) {
	case ast.TypeDefNode:
		tc.Defs[ident] = d
	case ast.TypeDefShapeExpr:
		tc.Defs[ident] = d
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
	// Look up the shape for this hash-based type
	hashDef, hashExists := tc.Defs[typeNode.Ident]
	if !hashExists {
		return ""
	}
	hashTypeDef, ok := hashDef.(ast.TypeDefNode)
	if !ok {
		return ""
	}
	hashShapeExpr, ok := hashTypeDef.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		return ""
	}
	hashShape := hashShapeExpr.Shape
	for _, def := range tc.Defs {
		userDef, ok := def.(ast.TypeDefNode)
		if !ok || userDef.Ident == "" || strings.HasPrefix(string(userDef.Ident), "T_") {
			continue
		}
		shapeExpr, ok := userDef.Expr.(ast.TypeDefShapeExpr)
		if !ok {
			continue
		}
		userShape := shapeExpr.Shape
		if tc.shapesAreStructurallyIdentical(hashShape, userShape) {
			return userDef.Ident
		}
	}
	return ""
}

// FindAnyStructurallyIdenticalNamedType returns the first user-defined named type that is structurally identical to the given shape, or "" if none.
// This function works for any shape, not just hash-based types.
func (tc *TypeChecker) FindAnyStructurallyIdenticalNamedType(shape ast.ShapeNode) ast.TypeIdent {
	// Check all user-defined types for structural identity
	for typeIdent, def := range tc.Defs {
		// Skip hash-based types
		if strings.HasPrefix(string(typeIdent), "T_") {
			continue
		}

		// Check if this is a type definition with a shape
		typeDef, ok := def.(ast.TypeDefNode)
		if !ok {
			continue
		}

		shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr)
		if !ok {
			continue
		}

		// Check if the shapes are structurally identical
		if tc.shapesAreStructurallyIdentical(shape, shapeExpr.Shape) {
			return typeIdent
		}
	}

	return ""
}
