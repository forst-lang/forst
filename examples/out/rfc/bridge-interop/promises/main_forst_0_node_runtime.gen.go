package main

import "encoding/json"
import "forst/bridgert"

const (
	forstNodeGenStepDone  = "done"
	forstNodeGenStepError = "error"
)

var forstBridgeManifestJSON string = "{\"version\":1,\"exports\":[{\"moduleId\":\"legacy/payment.ts\",\"name\":\"concurrentEcho\",\"kind\":\"asyncFunction\"},{\"moduleId\":\"legacy/payment.ts\",\"name\":\"create\",\"kind\":\"asyncFunction\"},{\"moduleId\":\"legacy/payment.ts\",\"name\":\"delayed\",\"kind\":\"asyncFunction\"},{\"moduleId\":\"legacy/payment.ts\",\"name\":\"failWithError\",\"kind\":\"asyncFunction\"},{\"moduleId\":\"legacy/payment.ts\",\"name\":\"failWithObject\",\"kind\":\"asyncFunction\"}]}"

func forst_node_callasync_legacy_payment_ts_concurrentEcho() (T_NTbLJjyksQg, error) {
	return bridgert.CallAsyncArgs[T_NTbLJjyksQg]("legacy/payment.ts", "concurrentEcho", json.RawMessage("[7]"))
}
func forst_node_callasync_legacy_payment_ts_create() (T_BSiWS9EsB18, error) {
	return bridgert.CallAsyncArgs[T_BSiWS9EsB18]("legacy/payment.ts", "create", json.RawMessage("[100,\"USD\"]"))
}
func init() {
	bridgert.MustConfigureFromManifest(forstBridgeManifestJSON)
}
