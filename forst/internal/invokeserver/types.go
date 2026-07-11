package invokeserver

import (
	"context"
	"encoding/json"

	"forst/internal/discovery"
	"forst/internal/invokedispatch"
)

// HTTPContractVersion is the normative dev HTTP API revision.
const HTTPContractVersion = "1"

// InvokeRequest is the POST /invoke body.
type InvokeRequest struct {
	Package   string          `json:"package"`
	Function  string          `json:"function"`
	Args      json.RawMessage `json:"args"`
	Streaming bool            `json:"streaming,omitempty"`
}

// Response is the JSON envelope for invoke HTTP endpoints.
type Response struct {
	Success bool            `json:"success"`
	Output  string          `json:"output,omitzero"`
	Error   string          `json:"error,omitzero"`
	Result  json.RawMessage `json:"result,omitzero"`
}

// VersionInfo is returned by GET /version.
type VersionInfo struct {
	Version         string `json:"version"`
	Commit          string `json:"commit"`
	Date            string `json:"date"`
	ContractVersion string `json:"contractVersion"`
	Runtime         string `json:"runtime,omitempty"`
}

// Config holds HTTP listener settings.
type Config struct {
	Host           string
	Port           string
	CORS           bool
	ReadTimeout    int
	WriteTimeout   int
	MaxRequestSize int64
	Runtime        string
}

// Addr returns host:port for Listen.
func (c Config) Addr() string {
	host := c.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := c.Port
	if port == "" {
		port = "8081"
	}
	return host + ":" + port
}

// BaseURL returns http://host:port for ready files and clients.
func (c Config) BaseURL() string {
	host := c.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := c.Port
	if port == "" {
		port = "8081"
	}
	return "http://" + host + ":" + port
}

// DispatchBackend executes invoke requests.
type DispatchBackend interface {
	Functions() map[string]map[string]discovery.FunctionInfo
	RefreshFunctions(ctx context.Context) error
	Invoke(ctx context.Context, pkg, fn string, args json.RawMessage) (*invokedispatch.InvokeResult, error)
	InvokeStream(ctx context.Context, pkg, fn string, args json.RawMessage) (<-chan invokedispatch.StreamChunk, error)
}
