//go:build !windows

package nodert

import (
	"syscall"
	"time"
)

// TerminateHostPID sends SIGTERM then SIGKILL to an external node host pid.
func TerminateHostPID(pid int, grace time.Duration) error {
	if pid <= 0 {
		return nil
	}
	if grace <= 0 {
		grace = shutdownGracePeriod
	}
	_ = syscall.Kill(pid, syscall.SIGTERM)
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); err != nil {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
		return err
	}
	for i := 0; i < 40; i++ {
		if err := syscall.Kill(pid, 0); err != nil {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}
