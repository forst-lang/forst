package main

import "encoding/json"
import "forst/nodert"

const (
	forstNodeGenStepDone  = "done"
	forstNodeGenStepError = "error"
)

var forstNodeManifestJSON string = "{\"version\":1,\"exports\":[{\"moduleId\":\"legacy/generators.ts\",\"name\":\"asyncNumbers\",\"kind\":\"asyncGenerator\"},{\"moduleId\":\"legacy/generators.ts\",\"name\":\"emptyGen\",\"kind\":\"generator\"},{\"moduleId\":\"legacy/generators.ts\",\"name\":\"finallyRunCount\",\"kind\":\"function\"},{\"moduleId\":\"legacy/generators.ts\",\"name\":\"resetFinallyCount\",\"kind\":\"function\"},{\"moduleId\":\"legacy/generators.ts\",\"name\":\"returnGen\",\"kind\":\"generator\"},{\"moduleId\":\"legacy/generators.ts\",\"name\":\"syncNumbers\",\"kind\":\"generator\"},{\"moduleId\":\"legacy/generators.ts\",\"name\":\"throwGen\",\"kind\":\"generator\"},{\"moduleId\":\"legacy/generators.ts\",\"name\":\"withFinally\",\"kind\":\"generator\"}]}"

type (
	forstNodeGenStep_float64 struct {
		Kind    string
		Message string
		Value   float64
	}
	forstNodeSeq_float64 struct {
		inner *nodert.Seq[float64]
	}
)

func (s *forstNodeSeq_float64) Close() {
	s.inner.Close()
}
func (s *forstNodeSeq_float64) NextBatch(maxItems int) ([]forstNodeGenStep_float64, error) {
	var raw, err = s.inner.NextBatch(maxItems)
	if err != nil {
		return nil, err
	}
	var out []forstNodeGenStep_float64
	for _, step := range raw {
		out = append(out, forstNodeGenStep_float64{Kind: string(step.Kind), Value: step.Value, Message: step.Message})
	}
	return out, nil
}
func forst_node_open_seq_legacy_generators_ts_asyncNumbers() (*forstNodeSeq_float64, error) {
	var seq, err = nodert.OpenSeqArgs[float64]("legacy/generators.ts", "asyncNumbers", nodert.ExportKindAsyncGenerator, json.RawMessage("[3]"))
	if err != nil {
		return nil, err
	}
	return &forstNodeSeq_float64{inner: seq}, nil
}
func forst_node_open_seq_legacy_generators_ts_emptyGen() (*forstNodeSeq_float64, error) {
	var seq, err = nodert.OpenSeqArgs[float64]("legacy/generators.ts", "emptyGen", nodert.ExportKindGenerator, json.RawMessage("[]"))
	if err != nil {
		return nil, err
	}
	return &forstNodeSeq_float64{inner: seq}, nil
}
func forst_node_open_seq_legacy_generators_ts_syncNumbers() (*forstNodeSeq_float64, error) {
	var seq, err = nodert.OpenSeqArgs[float64]("legacy/generators.ts", "syncNumbers", nodert.ExportKindGenerator, json.RawMessage("[3]"))
	if err != nil {
		return nil, err
	}
	return &forstNodeSeq_float64{inner: seq}, nil
}
func forst_node_open_seq_legacy_generators_ts_withFinally() (*forstNodeSeq_float64, error) {
	var seq, err = nodert.OpenSeqArgs[float64]("legacy/generators.ts", "withFinally", nodert.ExportKindGenerator, json.RawMessage("[]"))
	if err != nil {
		return nil, err
	}
	return &forstNodeSeq_float64{inner: seq}, nil
}
func init() {
	nodert.MustConfigureFromManifest(forstNodeManifestJSON)
}
