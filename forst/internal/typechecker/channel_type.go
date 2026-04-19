package typechecker

import "forst/internal/ast"

// ChannelElementType returns (T, true) when t is chan T in the Forst type IR.
func ChannelElementType(t ast.TypeNode) (ast.TypeNode, bool) {
	if t.Ident == ast.TypeChannel && len(t.TypeParams) >= 1 {
		return t.TypeParams[0], true
	}
	return ast.TypeNode{}, false
}

// IsChannelReturn is true when the function has a single return type chan T.
func IsChannelReturn(fn *ast.FunctionNode, tc *TypeChecker) bool {
	if fn == nil {
		return false
	}
	var rts []ast.TypeNode
	if tc != nil {
		if sig, ok := tc.Functions[fn.Ident.ID]; ok && len(sig.ReturnTypes) > 0 {
			rts = sig.ReturnTypes
		} else {
			rts = fn.ReturnTypes
		}
	} else {
		rts = fn.ReturnTypes
	}
	if len(rts) != 1 {
		return false
	}
	_, ok := ChannelElementType(rts[0])
	return ok
}
