package nodert

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Sentinel errors for node runtime client failures.
var (
	ErrNodeRuntimeDied    = errors.New("node runtime process exited")
	ErrForbidden          = errors.New("forbidden: module or export not in manifest")
	ErrNotInitialized     = errors.New("node runtime not initialized")
	ErrInitializeRequired = errors.New("initialize required before rpc call")
	ErrMethodNotFound     = errors.New("json-rpc method not found")
	ErrCallAsyncNotReady  = errors.New("forst.node/callAsync not implemented")
	ErrCallTimeout        = errors.New("node rpc call timed out")
	ErrInvalidModuleID    = errors.New("invalid moduleId")
)

// JSON-RPC error codes used by the node runtime protocol.
const (
	ErrCodeServerError    = -32000
	ErrCodeForbidden      = -32001
	ErrCodeMethodNotFound = -32601
)

// RPCError is a JSON-RPC error returned by the node runtime.
type RPCError struct {
	Code    int
	Message string
	Data    []byte
}

func (e *RPCError) Error() string {
	if e == nil {
		return "json-rpc error"
	}
	return fmt.Sprintf("json-rpc error %d: %s", e.Code, e.Message)
}

// IsForbidden reports whether err is or wraps ErrForbidden or JSON-RPC -32001.
func IsForbidden(err error) bool {
	if errors.Is(err, ErrForbidden) {
		return true
	}
	var rpcErr *RPCError
	if errors.As(err, &rpcErr) {
		return rpcErr.Code == ErrCodeForbidden
	}
	return false
}

// IsMethodNotFound reports whether err is or wraps ErrMethodNotFound or JSON-RPC -32601.
func IsMethodNotFound(err error) bool {
	if errors.Is(err, ErrMethodNotFound) {
		return true
	}
	var rpcErr *RPCError
	if errors.As(err, &rpcErr) {
		return rpcErr.Code == ErrCodeMethodNotFound
	}
	return false
}

func rpcErrorFromWire(code int, message string, data []byte) error {
	switch code {
	case ErrCodeForbidden:
		return fmt.Errorf("%w: %s", ErrForbidden, message)
	case ErrCodeMethodNotFound:
		return fmt.Errorf("%w: %s", ErrMethodNotFound, message)
	default:
		if nodeErr, ok := nodeCallErrorFromData(message, data); ok {
			return nodeErr
		}
		return &RPCError{Code: code, Message: message, Data: data}
	}
}

// NodeCallError is a structured error from a Node export throw or Promise rejection.
type NodeCallError struct {
	Message    string
	Name       string
	Stack      string
	ModuleID   string
	ExportName string
}

func (e *NodeCallError) Error() string {
	if e == nil {
		return "node call error"
	}
	if e.Stack != "" {
		return e.Message + "\n" + e.Stack
	}
	return e.Message
}

// AsNodeCallError reports whether err wraps a NodeCallError.
func AsNodeCallError(err error, target **NodeCallError) bool {
	if err == nil {
		return false
	}
	var nodeErr *NodeCallError
	if errors.As(err, &nodeErr) {
		*target = nodeErr
		return true
	}
	return false
}

func nodeCallErrorFromData(message string, data []byte) (*NodeCallError, bool) {
	if len(data) == 0 {
		return nil, false
	}
	var payload struct {
		Name       string `json:"name"`
		Stack      string `json:"stack"`
		ModuleID   string `json:"moduleId"`
		ExportName string `json:"exportName"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, false
	}
	if payload.Stack == "" && payload.ModuleID == "" && payload.ExportName == "" && payload.Name == "" {
		return nil, false
	}
	return &NodeCallError{
		Message:    message,
		Name:       payload.Name,
		Stack:      payload.Stack,
		ModuleID:   payload.ModuleID,
		ExportName: payload.ExportName,
	}, true
}
