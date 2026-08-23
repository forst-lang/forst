package typechecker

import "forst/internal/ast"

func (tc *TypeChecker) normalizeTypeForStorage(t ast.TypeNode) ast.TypeNode {
	switch t.StorageClass(tc.isBuiltinType) {
	case ast.TypeStorageTypeParam:
		return t
	case ast.TypeStorageBuiltinOrStructural:
		return t
	case ast.TypeStorageNamedUserType:
		return ensureUserDefinedType(t)
	default:
		return t
	}
}

func (tc *TypeChecker) normalizeTypesForStorage(types []ast.TypeNode) []ast.TypeNode {
	if len(types) == 0 {
		return types
	}
	out := make([]ast.TypeNode, len(types))
	for i, t := range types {
		out[i] = tc.normalizeTypeForStorage(t)
	}
	return out
}
