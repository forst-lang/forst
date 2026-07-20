//go:build windows

package nodert

import (
	"os"
	"time"
)

// TerminateHostPID kills an external node host pid. Windows has no POSIX
// signal semantics (no SIGTERM/SIGKILL distinction), so this issues a direct
// process kill and polls OpenProcess (via os.FindProcess) to detect exit.
func TerminateHostPID(pid int, grace time.Duration) error {
	if pid <= 0 {
		return nil
	}
	if grace <= 0 {
		grace = shutdownGracePeriod
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil
	}
	_ = proc.Kill()

	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if !windowsProcessAlive(pid) {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	for i := 0; i < 40; i++ {
		if !windowsProcessAlive(pid) {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}

func windowsProcessAlive(pid int) bool {
	_, err := os.FindProcess(pid)
	return err == nil
}
