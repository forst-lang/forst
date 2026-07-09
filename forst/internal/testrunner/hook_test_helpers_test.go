package testrunner

import (
	"sync/atomic"
	"testing"
)

// swapHook replaces a package-level hook for the duration of t, restoring it on cleanup.
// Uses atomic.Value so parallel tests and cleanup do not race under -race.
func swapHook[T any](t *testing.T, slot *atomic.Value, stub T) {
	t.Helper()
	orig := slot.Load()
	slot.Store(stub)
	t.Cleanup(func() { slot.Store(orig) })
}
