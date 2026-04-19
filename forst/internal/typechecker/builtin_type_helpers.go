package typechecker

import "forst/internal/ast"

// sliceElementType returns the element type of a slice/array TypeNode (Array(T)).
func sliceElementType(t ast.TypeNode) (ast.TypeNode, bool) {
	if t.Ident == ast.TypeArray && len(t.TypeParams) >= 1 {
		return t.TypeParams[0], true
	}
	return ast.TypeNode{}, false
}

// mapKeyValueTypes returns key and value types for Map(K, V).
func mapKeyValueTypes(t ast.TypeNode) (key, val ast.TypeNode, ok bool) {
	if t.Ident != ast.TypeMap || len(t.TypeParams) < 2 {
		return ast.TypeNode{}, ast.TypeNode{}, false
	}
	return t.TypeParams[0], t.TypeParams[1], true
}

// lenOperandAllowed mirrors Go's len: string, array, slice, map, pointer to array,
// and channel when modeled. Forst does not model channel types yet; pointer-to-slice
// uses the same Array(T) representation as a slice, so we allow Pointer(Array(T)).
//
// Opaque Go interop values (TYPE_IMPLICIT) are allowed: the operand may be a slice,
// string, map, etc. that we do not reify into a Forst type node; len() is still
// valid in Go and the generated code is checked by go/types.
func lenOperandAllowed(t ast.TypeNode) bool {
	switch t.Ident {
	case ast.TypeImplicit:
		return true
	case ast.TypeString, ast.TypeMap:
		return true
	case ast.TypeArray:
		return true
	case ast.TypePointer:
		if len(t.TypeParams) != 1 {
			return false
		}
		return t.TypeParams[0].Ident == ast.TypeArray
	default:
		return false
	}
}

// capOperandAllowed mirrors Go's cap: slice, array, channel.
// TYPE_IMPLICIT is allowed for the same reason as lenOperandAllowed.
func capOperandAllowed(t ast.TypeNode) bool {
	switch t.Ident {
	case ast.TypeImplicit:
		return true
	case ast.TypeChannel:
		return true
	case ast.TypeArray:
		return true
	case ast.TypePointer:
		if len(t.TypeParams) != 1 {
			return false
		}
		return t.TypeParams[0].Ident == ast.TypeArray
	default:
		return false
	}
}

// clearOperandAllowed: Go 1.21+ clear on map or slice.
func clearOperandAllowed(t ast.TypeNode) bool {
	return t.Ident == ast.TypeMap || t.Ident == ast.TypeArray
}

// builtinTypesIdenticalOrdered checks structural identity for min/max (same ordered type).
func builtinTypesIdenticalOrdered(a, b ast.TypeNode) bool {
	if a.Ident != b.Ident {
		return false
	}
	if len(a.TypeParams) != len(b.TypeParams) {
		return false
	}
	for i := range a.TypeParams {
		if !builtinTypesIdenticalOrdered(a.TypeParams[i], b.TypeParams[i]) {
			return false
		}
	}
	return true
}

// isOrderedBuiltinType is the subset of cmp.Ordered we model without full generics.
func isOrderedBuiltinType(t ast.TypeNode) bool {
	switch t.Ident {
	case ast.TypeInt, ast.TypeFloat, ast.TypeString:
		return true
	default:
		return false
	}
}
