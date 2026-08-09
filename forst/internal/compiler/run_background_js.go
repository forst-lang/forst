//go:build js

package compiler

import (
	"os"
	"syscall"
)

// newProcessGroupAttrs is a no-op on js/wasm: background process groups are unsupported.
func newProcessGroupAttrs() *syscall.SysProcAttr {
	return nil
}

// signalProcessGroup signals the process directly on js/wasm.
func signalProcessGroup(proc *os.Process, sig syscall.Signal) error {
	if proc == nil {
		return nil
	}
	return proc.Signal(sig)
}
