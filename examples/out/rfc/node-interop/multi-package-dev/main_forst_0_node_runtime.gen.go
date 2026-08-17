package main

import "encoding/json"
import "forst/nodert"

const (
	forstNodeGenStepDone  = "done"
	forstNodeGenStepError = "error"
)

var forstNodeManifestJSON string = "{\"version\":1,\"exports\":[{\"moduleId\":\"host.ts\",\"name\":\"hostPing\",\"kind\":\"function\"}]}"

func ForstNodeWaitForShutdown() {
	nodert.WaitForShutdown()
}
func forst_node_callsync_host_ts_hostPing() (string, error) {
	return nodert.CallSyncArgs[string]("host.ts", "hostPing", json.RawMessage("[]"))
}
func init() {
	nodert.MustConfigureFromManifest(forstNodeManifestJSON)
}
