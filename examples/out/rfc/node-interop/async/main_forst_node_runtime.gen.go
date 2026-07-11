package main

import "forst/nodert"

const (
	forstNodeGenStepDone  = "done"
	forstNodeGenStepError = "error"
)

var forstNodeManifestJSON string = "{\"version\":1,\"exports\":[{\"moduleId\":\"legacy/events.ts\",\"name\":\"dispatch\",\"kind\":\"asyncFunction\"},{\"moduleId\":\"legacy/events.ts\",\"name\":\"subscribe\",\"kind\":\"asyncGenerator\"},{\"moduleId\":\"legacy/payment.ts\",\"name\":\"concurrentEcho\",\"kind\":\"asyncFunction\"},{\"moduleId\":\"legacy/payment.ts\",\"name\":\"create\",\"kind\":\"asyncFunction\"},{\"moduleId\":\"legacy/payment.ts\",\"name\":\"delayed\",\"kind\":\"asyncFunction\"},{\"moduleId\":\"legacy/payment.ts\",\"name\":\"failWithError\",\"kind\":\"asyncFunction\"},{\"moduleId\":\"legacy/payment.ts\",\"name\":\"failWithObject\",\"kind\":\"asyncFunction\"}]}"

type (
	forstNodeGenStep_t_g2n6vmumv8n struct {
		Kind    string
		Message string
		Value   T_G2n6vMumV8n
	}
	forstNodeSeq_t_g2n6vmumv8n struct {
		inner nodert.Seq[T_G2n6vMumV8n]
	}
)

func (s *forstNodeSeq_t_g2n6vmumv8n) Close() {
	s.inner.Close()
}
func (s *forstNodeSeq_t_g2n6vmumv8n) NextBatch(maxItems int) ([]forstNodeGenStep_t_g2n6vmumv8n, error) {
	var raw, err = s.inner.NextBatch(maxItems)
	if err != nil {
		return nil, err
	}
	var out []forstNodeGenStep_t_g2n6vmumv8n
	for _, step := range raw {
		out = append(out, forstNodeGenStep_t_g2n6vmumv8n{Kind: string(step.Kind), Value: step.Value, Message: step.Message})
	}
	return out, nil
}
func forst_node_callasync_legacy_events_ts_dispatch(arg0 T_G2n6vMumV8n) (struct {
}, error) {
	return nodert.CallAsync[struct {
	}]("legacy/events.ts", "dispatch", arg0)
}
func forst_node_callasync_legacy_payment_ts_concurrentEcho(arg0 int) (T_NTbLJjyksQg, error) {
	return nodert.CallAsync[T_NTbLJjyksQg]("legacy/payment.ts", "concurrentEcho", arg0)
}
func forst_node_callasync_legacy_payment_ts_create(arg0 float64, arg1 string) (T_BSiWS9EsB18, error) {
	return nodert.CallAsync[T_BSiWS9EsB18]("legacy/payment.ts", "create", arg0, arg1)
}
func forst_node_open_seq_legacy_events_ts_subscribe(arg0 string) (*forstNodeSeq_t_g2n6vmumv8n, error) {
	var seq, err = nodert.OpenSeq[T_G2n6vMumV8n]("legacy/events.ts", "subscribe", nodert.ExportKindAsyncGenerator, arg0)
	if err != nil {
		return nil, err
	}
	return &forstNodeSeq_t_g2n6vmumv8n{inner: seq}, nil
}
func init() {
	nodert.MustConfigureFromManifest(forstNodeManifestJSON)
}
