//go:build js

package bridgert

import "syscall"

// hostSessionAttrs is a no-op on js/wasm: bridgert does not spawn Node hosts there.
func hostSessionAttrs() *syscall.SysProcAttr {
	return nil
}
