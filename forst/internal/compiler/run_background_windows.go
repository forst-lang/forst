//go:build windows

package compiler

import (
	"os"
	"syscall"
)

// newProcessGroupAttrs is a no-op on Windows: Setpgid is not a field on
// syscall.SysProcAttr for this platform.
func newProcessGroupAttrs() *syscall.SysProcAttr {
	return nil
}

// signalProcessGroup falls back to signaling the process directly: Windows
// has no POSIX process-group kill (syscall.Kill is undefined here).
func signalProcessGroup(proc *os.Process, sig syscall.Signal) error {
	if proc == nil {
		return nil
	}
	return proc.Signal(sig)
}
