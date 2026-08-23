package main

import "encoding/json"
import "forst/bridgert"

const (
	forstNodeGenStepDone  = "done"
	forstNodeGenStepError = "error"
)

var forstBridgeManifestJSON string = "{\"version\":1,\"exports\":[{\"moduleId\":\"legacy/counter.ts\",\"name\":\"inc\",\"kind\":\"function\"}]}"

func ForstNodeWaitForShutdown() {
	bridgert.WaitForShutdown()
}
func forst_node_callsync_legacy_counter_ts_inc() (float64, error) {
	return bridgert.CallSyncArgs[float64]("legacy/counter.ts", "inc", json.RawMessage("[]"))
}
func init() {
	bridgert.MustConfigureFromManifest(forstBridgeManifestJSON)
}
