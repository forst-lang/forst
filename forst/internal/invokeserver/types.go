package invokeserver

import (
	"encoding/json"
	"net"
	"strings"

	"forst/internal/forsterr"
)

// HTTPContractVersion is the normative dev HTTP API revision.
const HTTPContractVersion = "2"

// ContractVersionHTTPHeader is sent on every invoke HTTP response.
const ContractVersionHTTPHeader = "X-Forst-Contract-Version"

// Invoke auth HTTP headers (reserved; clients must not override).
const (
	HeaderInvokeProof      = "X-Forst-Invoke-Proof"
	HeaderInvokeGeneration = "X-Forst-Invoke-Generation"
	HeaderInvokeNonce      = "X-Forst-Invoke-Nonce"
	HeaderInvokeToken      = "X-Forst-Invoke-Token"
)

const (
	envInvokeAuthDisabled = "FORST_INVOKE_AUTH"
	transportTCP          = "tcp"
	transportUnix         = "unix"
)

// InvokeRequest is the POST /invoke body.
type InvokeRequest struct {
	Package   string          `json:"package"`
	Function  string          `json:"function"`
	Args      json.RawMessage `json:"args"`
	Streaming bool            `json:"streaming,omitempty"`
}

// ChallengeResponse is returned by GET /invoke/challenge.
type ChallengeResponse struct {
	Nonce      string `json:"nonce"`
	ExpiresAt  string `json:"expiresAt"`
	Generation uint64 `json:"generation"`
}

// ErrorValue is a structured nominal error on the invoke wire (contract v2).
type ErrorValue = forsterr.WireError

// Response is the JSON envelope for invoke HTTP endpoints.
type Response struct {
	Success    bool            `json:"success"`
	Output     string          `json:"output,omitzero"`
	Error      string          `json:"error,omitzero"`
	ErrorValue *ErrorValue     `json:"errorValue,omitempty"`
	Result     json.RawMessage `json:"result,omitzero"`
	Reloading  bool            `json:"reloading,omitempty"`
	Generation uint64          `json:"generation,omitempty"`
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
	Host                string
	Port                string
	SocketPath          string
	Transport           string
	// BoundaryRoot is the project root used for ready/token files and default socket paths.
	BoundaryRoot        string
	CORS                bool
	CORSAllowedOrigins  []string
	AllowedHosts        []string
	ReadTimeout         int
	WriteTimeout        int
	MaxRequestSize      int64
	MaxConcurrentInvoke int
	Runtime             string
	AllowNonLoopback    bool
	AuthDisabled        bool
}

func authDisabledByEnv() bool {
	v := strings.ToLower(strings.TrimSpace(envOrEmpty(envInvokeAuthDisabled)))
	return v == "off" || v == "0" || v == "false"
}

func envOrEmpty(key string) string {
	// small helper kept inline in types to avoid importing os in every test
	return lookupEnv(key)
}

func (c Config) listenHost() string {
	host := c.Host
	if c.Runtime == "embedded" {
		host = embeddedListenHost
	} else if host == "" {
		host = embeddedListenHost
	}
	if !c.AllowNonLoopback && !isLoopbackHost(host) {
		return embeddedListenHost
	}
	return host
}

// downgradedListenHost reports whether listenHost() replaced a configured non-loopback host.
func (c Config) downgradedListenHost() (requested, effective string, downgraded bool) {
	requested = strings.TrimSpace(c.Host)
	effective = c.listenHost()
	if requested == "" || c.AllowNonLoopback {
		return requested, effective, false
	}
	downgraded = !strings.EqualFold(requested, effective) && !isLoopbackHost(requested)
	return requested, effective, downgraded
}

func isLoopbackHost(host string) bool {
	host = strings.Trim(strings.ToLower(strings.TrimSpace(host)), "[]")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (c Config) listenPort() string {
	port := c.Port
	if port == "" {
		port = "8081"
	}
	return port
}

func (c Config) network() string {
	switch strings.ToLower(strings.TrimSpace(c.Transport)) {
	case transportUnix:
		return transportUnix
	default:
		return transportTCP
	}
}

// ListenTarget returns the socket path or host:port for net.Listen.
func (c Config) ListenTarget() string {
	if c.network() == transportUnix && c.SocketPath != "" {
		return c.SocketPath
	}
	return c.Addr()
}

// Addr returns host:port for Listen.
func (c Config) Addr() string {
	return c.listenHost() + ":" + c.listenPort()
}

// BaseURL returns http://host:port for ready files and clients.
func (c Config) BaseURL() string {
	return "http://" + c.listenHost() + ":" + c.listenPort()
}

func (c Config) authEnabled() bool {
	return !c.AuthDisabled && !authDisabledByEnv()
}
