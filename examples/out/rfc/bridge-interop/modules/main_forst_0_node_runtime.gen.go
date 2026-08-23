package main

import "encoding/json"
import "forst/bridgert"

const (
	forstNodeGenStepDone  = "done"
	forstNodeGenStepError = "error"
)

var forstBridgeManifestJSON string = "{\"version\":1,\"exports\":[{\"moduleId\":\"legacy/api/checkout.ts\",\"name\":\"createOrder\",\"kind\":\"function\"}]}"

func forst_node_callsync_legacy_api_checkout_ts_createOrder() (T_Zn4FXrBCht3, error) {
	return bridgert.CallSyncArgs[T_Zn4FXrBCht3]("legacy/api/checkout.ts", "createOrder", json.RawMessage("[100,\"USD\"]"))
}
func init() {
	bridgert.MustConfigureFromManifest(forstBridgeManifestJSON)
}
