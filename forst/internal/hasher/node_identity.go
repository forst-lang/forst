package hasher

import (
	"forst/internal/ast"
	"hash/fnv"
	"reflect"
	"unsafe"
)

// NodeIdentity is a (dynamic type, data pointer) pair for memoizing HashNode and scope lookup.
type NodeIdentity struct {
	Typ  uintptr
	Data uintptr
}

// NodeIdentityKey returns a stable identity for the concrete value in an ast.Node interface.
func NodeIdentityKey(node ast.Node) (NodeIdentity, bool) {
	if node == nil {
		return NodeIdentity{}, false
	}
	type iface struct {
		typ, data unsafe.Pointer
	}
	word := (*iface)(unsafe.Pointer(&node))
	if word.data == nil {
		return NodeIdentity{}, false
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(reflect.TypeOf(node).String()))
	return NodeIdentity{Typ: uintptr(h.Sum64()), Data: uintptr(word.data)}, true
}
