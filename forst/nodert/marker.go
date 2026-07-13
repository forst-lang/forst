package nodert

import (
	"fmt"
	"sync"
)

// ResetForTest clears supervisor and configure singleton state for integration tests.
func ResetForTest() {
	resetSupervisorForTest()
	configureOnce = sync.Once{}
	configureErr = nil
}

// ConfigureFromManifestForTest configures the supervisor from manifest JSON for tests.
func ConfigureFromManifestForTest(manifestJSON string) error {
	ResetForTest()
	return configureFromManifest(manifestJSON)
}

// CallSyncForTest invokes a synchronous Node export in tests.
func CallSyncForTest[T any](moduleID, export string) (T, error) {
	return CallSync[T](moduleID, export)
}

// ShutdownForTest shuts down the supervised Node process in tests.
func ShutdownForTest() error {
	return Shutdown()
}

// ReadHostMarkerPID returns the live node host pid from boundaryRoot/.forst/node.sock.ready.
// Returns 0 when the marker is missing or the pid is not alive.
func ReadHostMarkerPID(boundaryRoot string) int {
	_, readyPath, err := ResolveHostSocketPath(boundaryRoot, "")
	if err != nil {
		return 0
	}
	marker, ok := readHostReadyMarker(readyPath)
	if !ok || !processAlive(marker.PID) {
		return 0
	}
	return marker.PID
}

// ReattachSkipReason reports why an existing host cannot be reattached, or "" if reattach may proceed.
func ReattachSkipReason(readyPath string) string {
	marker, ok := readHostReadyMarker(readyPath)
	if !ok {
		return "marker_missing"
	}
	if !processAlive(marker.PID) {
		return fmt.Sprintf("pid_dead pid=%d", marker.PID)
	}
	if !hostMarkerReadyForRPC(marker) {
		return fmt.Sprintf("phase_not_ready phase=%q", marker.Phase)
	}
	return ""
}
