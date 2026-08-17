package main

import "encoding/json"
import "forst/nodert"

const (
	forstNodeGenStepDone  = "done"
	forstNodeGenStepError = "error"
)

var forstNodeManifestJSON string = "{\"version\":1,\"exports\":[{\"moduleId\":\"legacy/payment.ts\",\"name\":\"create\",\"kind\":\"function\"}]}"

func forst_node_callsync_legacy_payment_ts_create() (T_BSiWS9EsB18, error) {
	return nodert.CallSyncArgs[T_BSiWS9EsB18]("legacy/payment.ts", "create", json.RawMessage("[100,\"USD\"]"))
}
func init() {
	nodert.MustConfigureFromManifest(forstNodeManifestJSON)
}
