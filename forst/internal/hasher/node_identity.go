package hasher

import (
	"forst/internal/ast"
	"unsafe"
)

// NodeIdentity is a (dynamic type, data pointer) pair for memoizing HashNode and scope lookup.
type NodeIdentity struct {
	Typ  uintptr
	Data uintptr
}

type ifaceWords struct {
	typ, data unsafe.Pointer
}

// NodeIdentityKey returns a stable identity for the concrete value in an ast.Node interface.
// Typ is the itab pointer (unique per concrete type for ast.Node). Data is the boxed value pointer.
func NodeIdentityKey(node ast.Node) (NodeIdentity, bool) {
	if node == nil {
		return NodeIdentity{}, false
	}
	word := (*ifaceWords)(unsafe.Pointer(&node))
	if word.data == nil {
		return NodeIdentity{}, false
	}
	return NodeIdentity{Typ: uintptr(word.typ), Data: uintptr(word.data)}, true
}
