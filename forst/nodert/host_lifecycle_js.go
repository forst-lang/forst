//go:build js

package nodert

import "time"

// TerminateHostPID is a no-op on js/wasm: nodert does not manage host PIDs there.
func TerminateHostPID(pid int, grace time.Duration) error {
	return nil
}
