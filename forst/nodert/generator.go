package nodert

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
)

// GenStepKind is the pull-protocol step discriminator from forst.node/genNext.
type GenStepKind string

const (
	GenStepYield GenStepKind = "yield"
	GenStepDone  GenStepKind = "done"
	GenStepError GenStepKind = "error"
)

// GenStep is one iteration step from a Node generator stream.
type GenStep[T any] struct {
	Kind    GenStepKind `json:"kind"`
	Value   T           `json:"value,omitempty"`
	Message string      `json:"message,omitempty"`
}

// Seq is a blocking pull handle for a TypeScript generator export (sync or async).
type Seq[T any] struct {
	streamID string
	closed   bool
	mu       sync.Mutex
}

// Iterator is an alias for Seq (hand-written Go callers).
type Iterator[T any] = Seq[T]

// AsyncIterator is an alias for Seq (hand-written Go callers).
type AsyncIterator[T any] = Seq[T]

// GenOpenParams is sent with forst.node/genOpen.
type GenOpenParams struct {
	ModuleID   string          `json:"moduleId"`
	ExportName string          `json:"exportName"`
	Args       json.RawMessage `json:"args"`
}

// GenOpenResult is returned from forst.node/genOpen on success.
type GenOpenResult struct {
	StreamID string `json:"streamId"`
}

// GenNextParams is sent with forst.node/genNext.
type GenNextParams struct {
	StreamID string `json:"streamId"`
}

// GenNextBatchParams requests up to maxItems yields in one RPC.
type GenNextBatchParams struct {
	StreamID string `json:"streamId"`
	MaxItems int    `json:"maxItems"`
}

// GenCloseParams is sent with forst.node/genClose.
type GenCloseParams struct {
	StreamID string `json:"streamId"`
}

// GenCloseResult is returned from forst.node/genClose on success.
type GenCloseResult struct {
	OK bool `json:"ok"`
}

// OpenSeqArgs starts a generator stream with pre-marshaled JSON args (codegen fast path).
func OpenSeqArgs[T any](moduleID, exportName string, kind ExportKind, argsJSON json.RawMessage) (Seq[T], error) {
	streamID, err := openGenStreamJSON(moduleID, exportName, kind, argsJSON)
	if err != nil {
		return Seq[T]{}, err
	}
	it := Seq[T]{streamID: streamID}
	runtime.SetFinalizer(&it, (*Seq[T]).finalize)
	return it, nil
}

// OpenSeq starts a generator stream via forst.node/genOpen.
func OpenSeq[T any](moduleID, exportName string, kind ExportKind, args ...any) (Seq[T], error) {
	streamID, err := openGenStream(moduleID, exportName, kind, args...)
	if err != nil {
		return Seq[T]{}, err
	}
	it := Seq[T]{streamID: streamID}
	runtime.SetFinalizer(&it, (*Seq[T]).finalize)
	return it, nil
}

// OpenGen starts a sync generator stream.
func OpenGen[T any](moduleID, exportName string, args ...any) (Seq[T], error) {
	return OpenSeq[T](moduleID, exportName, ExportKindGenerator, args...)
}

// MustOpenGen is like OpenGen but panics on error (for hand-written Go callers).
func MustOpenGen[T any](moduleID, exportName string, args ...any) Seq[T] {
	it, err := OpenGen[T](moduleID, exportName, args...)
	if err != nil {
		panic(err)
	}
	return it
}

// OpenAsyncGen starts an async generator stream.
func OpenAsyncGen[T any](moduleID, exportName string, args ...any) (Seq[T], error) {
	return OpenSeq[T](moduleID, exportName, ExportKindAsyncGenerator, args...)
}

// MustOpenAsyncGen is like OpenAsyncGen but panics on error (for hand-written Go callers).
func MustOpenAsyncGen[T any](moduleID, exportName string, args ...any) Seq[T] {
	it, err := OpenAsyncGen[T](moduleID, exportName, args...)
	if err != nil {
		panic(err)
	}
	return it
}

func openGenStream(moduleID, exportName string, kind ExportKind, args ...any) (string, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("marshal gen open args: %w", err)
	}
	return openGenStreamJSON(moduleID, exportName, kind, argsJSON)
}

func openGenStreamJSON(moduleID, exportName string, kind ExportKind, argsJSON json.RawMessage) (string, error) {
	client, err := GetClient()
	if err != nil {
		return "", err
	}
	return client.GenOpen(moduleID, exportName, kind, argsJSON)
}

// Next pulls the next yield, done, or error step from Node.
func (it *Seq[T]) Next() (GenStep[T], error) {
	it.mu.Lock()
	defer it.mu.Unlock()
	if it.closed || it.streamID == "" {
		return GenStep[T]{Kind: GenStepDone}, nil
	}
	client, err := GetClient()
	if err != nil {
		return GenStep[T]{}, err
	}
	return clientGenNext[T](client, it.streamID)
}

// NextBatch pulls up to maxItems steps in one RPC when the client supports batching.
func (it *Seq[T]) NextBatch(maxItems int) ([]GenStep[T], error) {
	it.mu.Lock()
	defer it.mu.Unlock()
	if it.closed || it.streamID == "" {
		return []GenStep[T]{{Kind: GenStepDone}}, nil
	}
	if maxItems <= 0 {
		maxItems = 1
	}
	client, err := GetClient()
	if err != nil {
		return nil, err
	}
	return clientGenNextBatch[T](client, it.streamID, maxItems)
}

// Close releases the stream idempotently (forst.node/genClose).
func (it *Seq[T]) Close() error {
	it.mu.Lock()
	defer it.mu.Unlock()
	if it.closed || it.streamID == "" {
		return nil
	}
	client, err := GetClient()
	if err != nil {
		it.closed = true
		it.streamID = ""
		return err
	}
	streamID := it.streamID
	it.streamID = ""
	it.closed = true
	runtime.SetFinalizer(it, nil)
	return client.GenClose(streamID)
}

func (it *Seq[T]) finalize() {
	_ = it.Close()
}

// GenOpen invokes forst.node/genOpen after manifest preflight.
func (c *Client) GenOpen(moduleID, exportName string, kind ExportKind, args json.RawMessage) (string, error) {
	c.mu.Lock()
	manifest := c.manifest
	c.mu.Unlock()
	if err := manifest.AllowCall(moduleID, exportName, kind); err != nil {
		return "", err
	}
	params := GenOpenParams{
		ModuleID:   moduleID,
		ExportName: exportName,
		Args:       args,
	}
	result, err := c.Call(MethodGenOpen, params)
	if err != nil {
		return "", err
	}
	var open GenOpenResult
	if err := json.Unmarshal(result, &open); err != nil {
		return "", fmt.Errorf("decode genOpen result: %w", err)
	}
	if open.StreamID == "" {
		return "", fmt.Errorf("genOpen: empty streamId")
	}
	return open.StreamID, nil
}

// clientGenNext invokes forst.node/genNext and decodes the typed step.
func clientGenNext[T any](c *Client, streamID string) (GenStep[T], error) {
	steps, err := clientGenNextBatch[T](c, streamID, 1)
	if err != nil {
		return GenStep[T]{}, err
	}
	if len(steps) == 0 {
		return GenStep[T]{Kind: GenStepDone}, nil
	}
	return steps[0], nil
}

func clientGenNextBatch[T any](c *Client, streamID string, maxItems int) ([]GenStep[T], error) {
	if maxItems <= 0 {
		maxItems = 1
	}
	result, err := c.Call(MethodGenNextBatch, GenNextBatchParams{StreamID: streamID, MaxItems: maxItems})
	if err != nil {
		// Fallback for v1 runtimes without batch RPC.
		if maxItems == 1 {
			return decodeSingleGenNext[T](c, streamID)
		}
		return nil, err
	}
	var wire struct {
		Steps []struct {
			Kind    GenStepKind     `json:"kind"`
			Value   json.RawMessage `json:"value"`
			Message string          `json:"message"`
			Data    json.RawMessage `json:"data"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(result, &wire); err != nil {
		if maxItems == 1 {
			return decodeSingleGenNext[T](c, streamID)
		}
		return nil, fmt.Errorf("decode genNextBatch result: %w", err)
	}
	out := make([]GenStep[T], 0, len(wire.Steps))
	for _, s := range wire.Steps {
		step, err := decodeGenStepWire[T](s.Kind, s.Value, s.Message, s.Data)
		if err != nil {
			return nil, err
		}
		out = append(out, step)
	}
	return out, nil
}

func decodeSingleGenNext[T any](c *Client, streamID string) ([]GenStep[T], error) {
	result, err := c.Call(MethodGenNext, GenNextParams{StreamID: streamID})
	if err != nil {
		return nil, err
	}
	var wire struct {
		Kind    GenStepKind     `json:"kind"`
		Value   json.RawMessage `json:"value"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(result, &wire); err != nil {
		return nil, fmt.Errorf("decode genNext result: %w", err)
	}
	step, err := decodeGenStepWire[T](wire.Kind, wire.Value, wire.Message, wire.Data)
	if err != nil {
		return nil, err
	}
	return []GenStep[T]{step}, nil
}

func decodeGenStepWire[T any](kind GenStepKind, value json.RawMessage, message string, data json.RawMessage) (GenStep[T], error) {
	var zero GenStep[T]
	step := GenStep[T]{Kind: kind, Message: message}
	switch kind {
	case GenStepYield, GenStepDone:
		if len(value) > 0 && string(value) != "null" {
			if err := json.Unmarshal(value, &step.Value); err != nil {
				return zero, fmt.Errorf("decode genNext value: %w", err)
			}
		}
	case GenStepError:
		if step.Message == "" {
			step.Message = "generator step error"
		}
		if len(data) > 0 {
			return zero, &RPCError{
				Code:    ErrCodeServerError,
				Message: step.Message,
				Data:    data,
			}
		}
	default:
		return zero, fmt.Errorf("genNext: unknown kind %q", kind)
	}
	return step, nil
}

// GenClose invokes forst.node/genClose.
func (c *Client) GenClose(streamID string) error {
	_, err := c.Call(MethodGenClose, GenCloseParams{StreamID: streamID})
	return err
}
