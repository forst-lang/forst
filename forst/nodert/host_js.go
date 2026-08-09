//go:build js

package nodert

import "syscall"

// hostSessionAttrs is a no-op on js/wasm: nodert does not spawn Node hosts there.
func hostSessionAttrs() *syscall.SysProcAttr {
	return nil
}
