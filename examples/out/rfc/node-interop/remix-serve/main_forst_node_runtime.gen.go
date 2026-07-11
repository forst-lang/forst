package main

import "encoding/json"
import "forst/nodert"

const (
	forstNodeGenStepDone  = "done"
	forstNodeGenStepError = "error"
)

var forstNodeManifestJSON string = "{\"version\":1,\"exports\":[{\"moduleId\":\"legacy/activity.ts\",\"name\":\"activityFeed\",\"kind\":\"asyncGenerator\"},{\"moduleId\":\"legacy/activity.ts\",\"name\":\"dispatchActivity\",\"kind\":\"asyncFunction\"},{\"moduleId\":\"legacy/activity.ts\",\"name\":\"recentTitles\",\"kind\":\"generator\"},{\"moduleId\":\"legacy/todos.ts\",\"name\":\"addTodo\",\"kind\":\"function\"},{\"moduleId\":\"legacy/todos.ts\",\"name\":\"allTodos\",\"kind\":\"generator\"},{\"moduleId\":\"legacy/todos.ts\",\"name\":\"bumpEditCount\",\"kind\":\"function\"},{\"moduleId\":\"legacy/todos.ts\",\"name\":\"formatTodoList\",\"kind\":\"function\"},{\"moduleId\":\"legacy/todos.ts\",\"name\":\"openCount\",\"kind\":\"function\"},{\"moduleId\":\"legacy/todos.ts\",\"name\":\"persistSnapshot\",\"kind\":\"asyncFunction\"},{\"moduleId\":\"legacy/todos.ts\",\"name\":\"todoCount\",\"kind\":\"function\"},{\"moduleId\":\"legacy/todos.ts\",\"name\":\"toggleTodo\",\"kind\":\"function\"}]}"

type (
	forstNodeGenStep_string struct {
		Kind    string
		Message string
		Value   string
	}
	forstNodeSeq_string struct {
		inner *nodert.Seq[string]
	}
)
type (
	forstNodeGenStep_t_lkhz7dyfnqt struct {
		Kind    string
		Message string
		Value   T_LKhz7DyfNqT
	}
	forstNodeSeq_t_lkhz7dyfnqt struct {
		inner *nodert.Seq[T_LKhz7DyfNqT]
	}
)

func (s *forstNodeSeq_string) Close() {
	s.inner.Close()
}
func (s *forstNodeSeq_t_lkhz7dyfnqt) Close() {
	s.inner.Close()
}
func (s *forstNodeSeq_t_lkhz7dyfnqt) NextBatch(maxItems int) ([]forstNodeGenStep_t_lkhz7dyfnqt, error) {
	var raw, err = s.inner.NextBatch(maxItems)
	if err != nil {
		return nil, err
	}
	var out []forstNodeGenStep_t_lkhz7dyfnqt
	for _, step := range raw {
		out = append(out, forstNodeGenStep_t_lkhz7dyfnqt{Kind: string(step.Kind), Value: step.Value, Message: step.Message})
	}
	return out, nil
}
func (s *forstNodeSeq_string) NextBatch(maxItems int) ([]forstNodeGenStep_string, error) {
	var raw, err = s.inner.NextBatch(maxItems)
	if err != nil {
		return nil, err
	}
	var out []forstNodeGenStep_string
	for _, step := range raw {
		out = append(out, forstNodeGenStep_string{Kind: string(step.Kind), Value: step.Value, Message: step.Message})
	}
	return out, nil
}
func forst_node_callasync_legacy_activity_ts_dispatchActivity(arg0 T_LKhz7DyfNqT) (struct {
}, error) {
	return nodert.CallAsync[struct {
	}]("legacy/activity.ts", "dispatchActivity", arg0)
}
func forst_node_callasync_legacy_todos_ts_persistSnapshot() (T_8ycLsMp1YzS, error) {
	return nodert.CallAsyncArgs[T_8ycLsMp1YzS]("legacy/todos.ts", "persistSnapshot", json.RawMessage("[]"))
}
func forst_node_callsync_legacy_todos_ts_addTodo(arg0 string) (T_KuaRmDfgFpc, error) {
	return nodert.CallSync[T_KuaRmDfgFpc]("legacy/todos.ts", "addTodo", arg0)
}
func forst_node_callsync_legacy_todos_ts_bumpEditCount() (float64, error) {
	return nodert.CallSyncArgs[float64]("legacy/todos.ts", "bumpEditCount", json.RawMessage("[]"))
}
func forst_node_callsync_legacy_todos_ts_formatTodoList() (string, error) {
	return nodert.CallSyncArgs[string]("legacy/todos.ts", "formatTodoList", json.RawMessage("[]"))
}
func forst_node_callsync_legacy_todos_ts_openCount() (float64, error) {
	return nodert.CallSyncArgs[float64]("legacy/todos.ts", "openCount", json.RawMessage("[]"))
}
func forst_node_callsync_legacy_todos_ts_todoCount() (float64, error) {
	return nodert.CallSyncArgs[float64]("legacy/todos.ts", "todoCount", json.RawMessage("[]"))
}
func forst_node_callsync_legacy_todos_ts_toggleTodo(arg0 string) (T_KuaRmDfgFpc, error) {
	return nodert.CallSync[T_KuaRmDfgFpc]("legacy/todos.ts", "toggleTodo", arg0)
}
func forst_node_open_seq_legacy_activity_ts_activityFeed() (*forstNodeSeq_t_lkhz7dyfnqt, error) {
	var seq, err = nodert.OpenSeqArgs[T_LKhz7DyfNqT]("legacy/activity.ts", "activityFeed", nodert.ExportKindAsyncGenerator, json.RawMessage("[\"demo\"]"))
	if err != nil {
		return nil, err
	}
	return &forstNodeSeq_t_lkhz7dyfnqt{inner: seq}, nil
}
func forst_node_open_seq_legacy_activity_ts_recentTitles() (*forstNodeSeq_string, error) {
	var seq, err = nodert.OpenSeqArgs[string]("legacy/activity.ts", "recentTitles", nodert.ExportKindGenerator, json.RawMessage("[3]"))
	if err != nil {
		return nil, err
	}
	return &forstNodeSeq_string{inner: seq}, nil
}
func init() {
	nodert.MustConfigureFromManifest(forstNodeManifestJSON)
}
