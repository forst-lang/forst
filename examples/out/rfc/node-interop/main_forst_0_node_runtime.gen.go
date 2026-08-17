package main

import "encoding/json"
import "forst/nodert"

const (
	forstNodeGenStepDone  = "done"
	forstNodeGenStepError = "error"
)

var forstNodeManifestJSON string = "{\"version\":1,\"exports\":[{\"moduleId\":\"legacy/payment.ts\",\"name\":\"create\",\"kind\":\"function\"}]}"

func forst_node_callsync_legacy_payment_ts_create() (T_S47SAU5d2zT, error) {
	return nodert.CallSyncArgs[T_S47SAU5d2zT]("legacy/payment.ts", "create", json.RawMessage("[100,\"USD\"]"))
}
func init() {
	nodert.MustConfigureFromManifest(forstNodeManifestJSON)
}
