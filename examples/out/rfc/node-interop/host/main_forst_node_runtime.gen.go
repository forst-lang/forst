package main

import "encoding/json"
import "forst/nodert"

const (
	forstNodeGenStepDone  = "done"
	forstNodeGenStepError = "error"
)

var forstNodeManifestJSON string = "{\"version\":1,\"exports\":[{\"moduleId\":\"legacy/counter.ts\",\"name\":\"inc\",\"kind\":\"function\"}]}"

func forst_node_callsync_legacy_counter_ts_inc() (float64, error) {
	return nodert.CallSyncArgs[float64]("legacy/counter.ts", "inc", json.RawMessage("[]"))
}
func init() {
	nodert.MustConfigureFromManifest(forstNodeManifestJSON)
}
