package main

import "strconv"

type T_BSiWS9EsB18 struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Id       string  `json:"id"`
}
type T_G2n6vMumV8n struct {
	Type string `json:"type"`
}
type T_NTbLJjyksQg struct {
	Echo float64 `json:"echo"`
}

func checkout(amount float64, currency string) string {
	result, resultErr := forst_node_callasync_legacy_payment_ts_create(amount, currency)
	if !(resultErr == nil) {
		return ""
	}
	return result.Id
}
func drainEvents(userId string) int {
	seq, seqErr := forst_node_open_seq_legacy_events_ts_subscribe(userId)
	if !(seqErr == nil) {
		return 0
	}
	var count int = 0
	{
		_nodeIt := seq
		defer _nodeIt.Close()
		var (
			_nodeStep     forstNodeGenStep_t_g2n6vmumv8n
			_nodeBatch    []forstNodeGenStep_t_g2n6vmumv8n
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
			forst_node_callasync_legacy_events_ts_dispatch(evt)
			count = count + 1
		}
	}
	return count
}
func echo(n float64) string {
	res, resErr := forst_node_callasync_legacy_payment_ts_concurrentEcho(n)
	if !(resErr == nil) {
		return ""
	}
	return strconv.FormatFloat(res.Echo, 'f', 0, 64)
}
func main() {
	id := checkout(100, "USD")
	println("payment:" + id)
	msg := echo(7)
	println("echo:" + msg)
	n := drainEvents("user-42")
	println("events:" + strconv.Itoa(n))
}
