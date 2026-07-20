//go:build !windows

package compiler

import (
	"os"
	"syscall"
)

// newProcessGroupAttrs puts the go run/built binary child in its own process
// group so signalProcessGroup can stop it (and any grandchildren) together.
func newProcessGroupAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

func signalProcessGroup(proc *os.Process, sig syscall.Signal) error {
	if proc == nil {
		return nil
	}
	if err := syscall.Kill(-proc.Pid, sig); err != nil {
		return proc.Signal(sig)
	}
	return nil
}
