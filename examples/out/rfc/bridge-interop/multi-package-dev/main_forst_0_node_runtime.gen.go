package main

import "encoding/json"
import "forst/bridgert"

const (
	forstNodeGenStepDone  = "done"
	forstNodeGenStepError = "error"
)

var forstBridgeManifestJSON string = "{\"version\":1,\"exports\":[{\"moduleId\":\"host.ts\",\"name\":\"hostPing\",\"kind\":\"function\"}]}"

func ForstNodeWaitForShutdown() {
	bridgert.WaitForShutdown()
}
func forst_node_callsync_host_ts_hostPing() (string, error) {
	return bridgert.CallSyncArgs[string]("host.ts", "hostPing", json.RawMessage("[]"))
}
func init() {
	bridgert.MustConfigureFromManifest(forstBridgeManifestJSON)
}
