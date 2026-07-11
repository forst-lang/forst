package main

import "strconv"
import os "os"
// AddTodoRequest: TypeDefShapeExpr({title: String})
type AddTodoRequest struct {
	Title string `json:"title"`
}
// AddTodoResponse: TypeDefShapeExpr({id: String, title: String, status: String})
type AddTodoResponse struct {
	Id     string `json:"id"`
	Status string `json:"status"`
	Title  string `json:"title"`
}
// CompleteTodoRequest: TypeDefShapeExpr({id: String})
type CompleteTodoRequest struct {
	Id string `json:"id"`
}
// CompleteTodoResponse: TypeDefShapeExpr({id: String, title: String, status: String})
type CompleteTodoResponse struct {
	Id     string `json:"id"`
	Status string `json:"status"`
	Title  string `json:"title"`
}
// DashboardResponse: TypeDefShapeExpr({open: Int, recentTitles: String, activityKinds: String, savedAt: String})
type DashboardResponse struct {
	ActivityKinds string `json:"activityKinds"`
	Open          int    `json:"open"`
	RecentTitles  string `json:"recentTitles"`
	SavedAt       string `json:"savedAt"`
}
// ListTodosResponse: TypeDefShapeExpr({open: Int, done: Int, encoded: String})
type ListTodosResponse struct {
	Done    int    `json:"done"`
	Encoded string `json:"encoded"`
	Open    int    `json:"open"`
}
// T_7nWLvcjQ76D: TypeDefShapeExpr({activityKinds: Value("ready"), open: Value(Variable(open)), recentTitles: Value(""), savedAt: Value(Variable(snap.savedAt))})
type T_7nWLvcjQ76D struct {
	ActivityKinds string  `json:"activityKinds"`
	Open          float64 `json:"open"`
	RecentTitles  string  `json:"recentTitles"`
	SavedAt       string  `json:"savedAt"`
}
type T_8ycLsMp1YzS struct {
	SavedAt string `json:"savedAt"`
}
// T_D415raHQ7uQ: TypeDefShapeExpr({done: Value(Variable(done)), encoded: Value(Variable(encoded)), open: Value(Variable(open))})
type T_D415raHQ7uQ struct {
	Done    float64 `json:"done"`
	Encoded string  `json:"encoded"`
	Open    float64 `json:"open"`
}
type T_KuaRmDfgFpc struct {
	Id     string `json:"id"`
	Status string `json:"status"`
	Title  string `json:"title"`
}
type T_LKhz7DyfNqT struct {
	Kind string `json:"kind"`
}

func AddTodo(input AddTodoRequest) AddTodoResponse {
	created, createdErr := forst_node_callsync_legacy_todos_ts_addTodo(input.Title)
	if !(createdErr == nil) {
		return AddTodoResponse{Id: "", Title: "", Status: ""}
	}
	return AddTodoResponse{Id: created.Id, Title: created.Title, Status: created.Status}
}
func CompleteTodo(input CompleteTodoRequest) AddTodoResponse {
	updated, updatedErr := forst_node_callsync_legacy_todos_ts_toggleTodo(input.Id)
	if !(updatedErr == nil) {
		return AddTodoResponse{Id: "", Title: "", Status: ""}
	}
	return AddTodoResponse{Id: updated.Id, Title: updated.Title, Status: updated.Status}
}
func GetDashboard() T_7nWLvcjQ76D {
	open, openErr := forst_node_callsync_legacy_todos_ts_openCount()
	if !(openErr == nil) {
		return T_7nWLvcjQ76D{Open: 0.0, RecentTitles: "", ActivityKinds: "", SavedAt: ""}
	}
	snap, snapErr := forst_node_callasync_legacy_todos_ts_persistSnapshot()
	if !(snapErr == nil) {
		return T_7nWLvcjQ76D{Open: 0.0, RecentTitles: "", ActivityKinds: "", SavedAt: ""}
	}
	return T_7nWLvcjQ76D{Open: open, RecentTitles: "", ActivityKinds: "ready", SavedAt: snap.SavedAt}
}
func ListTodos() T_D415raHQ7uQ {
	encoded, encodedErr := forst_node_callsync_legacy_todos_ts_formatTodoList()
	if !(encodedErr == nil) {
		return T_D415raHQ7uQ{Open: 0.0, Done: 0.0, Encoded: ""}
	}
	open, openErr := forst_node_callsync_legacy_todos_ts_openCount()
	if !(openErr == nil) {
		return T_D415raHQ7uQ{Open: 0.0, Done: 0.0, Encoded: ""}
	}
	total, totalErr := forst_node_callsync_legacy_todos_ts_todoCount()
	if !(totalErr == nil) {
		return T_D415raHQ7uQ{Open: 0.0, Done: 0.0, Encoded: ""}
	}
	done := total - open
	return T_D415raHQ7uQ{Open: open, Done: done, Encoded: encoded}
}
func main() {
	first, firstErr := forst_node_callsync_legacy_todos_ts_bumpEditCount()
	if !(firstErr == nil) {
		os.Exit(1)
	}
	println("sync:" + strconv.FormatFloat(first, 'f', 0, 64))
	second, secondErr := forst_node_callsync_legacy_todos_ts_bumpEditCount()
	if !(secondErr == nil) {
		os.Exit(1)
	}
	println("sync:" + strconv.FormatFloat(second, 'f', 0, 64))
	snap, snapErr := forst_node_callasync_legacy_todos_ts_persistSnapshot()
	if !(snapErr == nil) {
		os.Exit(1)
	}
	println("async:" + snap.SavedAt)
	var titleCount int = 0
	titleSeq, titleSeqErr := forst_node_open_seq_legacy_activity_ts_recentTitles()
	if !(titleSeqErr == nil) {
		os.Exit(1)
	}
	{
		_nodeIt := titleSeq
		defer _nodeIt.Close()
		var (
			_nodeStep     forstNodeGenStep_string
			_nodeBatch    []forstNodeGenStep_string
			_nodeBatchIdx int
			_nodeBatchErr error
		)
		_nodeBatch, _nodeBatchErr = _nodeIt.NextBatch(32)
		if _nodeBatchErr != nil {
			panic(_nodeBatchErr)
		}
		_nodeBatchIdx = 0
		for {
			if _nodeBatchIdx >= len(_nodeBatch) {
				_nodeBatch, _nodeBatchErr = _nodeIt.NextBatch(32)
				if _nodeBatchErr != nil {
					panic(_nodeBatchErr)
				}
				_nodeBatchIdx = 0
			}
			_nodeStep = _nodeBatch[_nodeBatchIdx]
			_nodeBatchIdx++
			if _nodeStep.Kind == forstNodeGenStepDone {
				break
			}
			if _nodeStep.Kind == forstNodeGenStepError {
				panic(_nodeStep.Message)
			}
			titleCount = titleCount + 1
		}
	}
	println("gen:" + strconv.Itoa(titleCount))
	var feedCount int = 0
	feed, feedErr := forst_node_open_seq_legacy_activity_ts_activityFeed()
	if !(feedErr == nil) {
		os.Exit(1)
	}
	{
		_nodeIt := feed
		defer _nodeIt.Close()
		var (
			_nodeStep     forstNodeGenStep_t_lkhz7dyfnqt
			_nodeBatch    []forstNodeGenStep_t_lkhz7dyfnqt
			_nodeBatchIdx int
			_nodeBatchErr error
		)
		_nodeBatch, _nodeBatchErr = _nodeIt.NextBatch(32)
		if _nodeBatchErr != nil {
			panic(_nodeBatchErr)
		}
		_nodeBatchIdx = 0
		for {
			if _nodeBatchIdx >= len(_nodeBatch) {
				_nodeBatch, _nodeBatchErr = _nodeIt.NextBatch(32)
				if _nodeBatchErr != nil {
					panic(_nodeBatchErr)
				}
				_nodeBatchIdx = 0
			}
			_nodeStep = _nodeBatch[_nodeBatchIdx]
			_nodeBatchIdx++
			if _nodeStep.Kind == forstNodeGenStepDone {
				break
			}
			if _nodeStep.Kind == forstNodeGenStepError {
				panic(_nodeStep.Message)
			}
			evt := _nodeStep.Value
			forst_node_callasync_legacy_activity_ts_dispatchActivity(evt)
			feedCount = feedCount + 1
		}
	}
	println("events:" + strconv.Itoa(feedCount))
	ForstInvokeWaitForShutdown()
}
