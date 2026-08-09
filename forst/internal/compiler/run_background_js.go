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

// signalProcessGroup is a no-op on js/wasm. OS signals cannot cancel child
// processes here; GoProgramProcess.Stop waits on the done channel instead.
func signalProcessGroup(proc *os.Process, sig syscall.Signal) error {
	if proc == nil {
		return nil
	}
	return nil
}
