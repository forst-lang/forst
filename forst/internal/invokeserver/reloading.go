package invokeserver

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type reloadMarkerPayload struct {
	Reloading  bool   `json:"reloading"`
	Generation uint64 `json:"generation"`
}

// ReadReloadMarker reports whether boundaryRoot/.forst/reloading is active.
func ReadReloadMarker(boundaryRoot string) (reloading bool, generation uint64) {
	if boundaryRoot == "" {
		return false, 0
	}
	path := filepath.Join(filepath.Clean(boundaryRoot), ".forst", "reloading")
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, 0
	}
	var payload reloadMarkerPayload
	if json.Unmarshal(raw, &payload) != nil {
		return false, 0
	}
	return payload.Reloading, payload.Generation
}
