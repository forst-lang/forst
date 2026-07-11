package nodert

import "encoding/json"

const (
	JSONRPCVersion  = "2.0"
	ProtocolVersion = 1
	ManifestVersion = 1
	DefaultMaxMsgLen = 16 << 20 // 16 MiB
)

// RPC method names (closed set for Phase 2+).
const (
	MethodInitialize = "forst.node/initialize"
	MethodPing       = "forst.node/ping"
	MethodCall       = "forst.node/call"
	MethodCallAsync  = "forst.node/callAsync"
	MethodGenOpen    = "forst.node/genOpen"
	MethodGenNext      = "forst.node/genNext"
	MethodGenNextBatch = "forst.node/genNextBatch"
	MethodGenClose     = "forst.node/genClose"
	MethodShutdown   = "forst.node/shutdown"
)

// Request is a JSON-RPC 2.0 request object.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response object.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ResponseError  `json:"error,omitempty"`
}

// ResponseError is the error object in a JSON-RPC response.
type ResponseError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// InitializeParams is sent with forst.node/initialize.
type InitializeParams struct {
	ProtocolVersion    int      `json:"protocolVersion"`
	BoundaryRoot       string   `json:"boundaryRoot"`
	Manifest           Manifest `json:"manifest"`
	FilesExclude       []string `json:"filesExclude,omitempty"`
	SupportedProtocols []string `json:"supportedProtocols,omitempty"`
}

// InitializeResult is returned from forst.node/initialize.
type InitializeResult struct {
	OK       bool   `json:"ok,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

// CallParams is sent with forst.node/call.
type CallParams struct {
	ModuleID   string          `json:"moduleId"`
	ExportName string          `json:"exportName"`
	Args       json.RawMessage `json:"args"`
}

// CallResult is returned from forst.node/call on success.
type CallResult struct {
	Value json.RawMessage `json:"value"`
}

// PingResult is returned from forst.node/ping on success.
type PingResult struct {
	OK bool `json:"ok"`
}

// ShutdownResult is returned from forst.node/shutdown on success.
type ShutdownResult struct {
	OK bool `json:"ok"`
}
