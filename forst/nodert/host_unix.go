//go:build !windows

package nodert

import "syscall"

// hostSessionAttrs puts the Node host in its own session so forst dev reload
// (process-group stop on go run) does not kill Vite/Remix children.
func hostSessionAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
