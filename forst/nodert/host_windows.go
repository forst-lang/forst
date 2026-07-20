//go:build windows

package nodert

import "syscall"

// hostSessionAttrs is a no-op on Windows: there is no POSIX session concept,
// and Setsid is not a field on syscall.SysProcAttr for this platform.
func hostSessionAttrs() *syscall.SysProcAttr {
	return nil
}
