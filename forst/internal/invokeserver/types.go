package invokeserver

import (
	"encoding/json"
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

// embeddedListenHost is the only bind address for embedded node-to-forst RPC.
const embeddedListenHost = "127.0.0.1"

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

func (c Config) listenHost() string {
	if c.Runtime == "embedded" {
		return embeddedListenHost
	}
	if c.Host == "" {
		return embeddedListenHost
	}
	return c.Host
}

func (c Config) listenPort() string {
	port := c.Port
	if port == "" {
		port = "8081"
	}
	return port
}

// Addr returns host:port for Listen.
func (c Config) Addr() string {
	return c.listenHost() + ":" + c.listenPort()
}

// BaseURL returns http://host:port for ready files and clients.
func (c Config) BaseURL() string {
	return "http://" + c.listenHost() + ":" + c.listenPort()
}
