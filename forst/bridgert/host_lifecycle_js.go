//go:build js

package bridgert

import "time"

// TerminateHostPID is a no-op on js/wasm: bridgert does not manage host PIDs there.
func TerminateHostPID(pid int, grace time.Duration) error {
	return nil
}
