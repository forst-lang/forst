// Package devserver implements the HTTP API for forst dev (invoke, types, health, etc.).
package devserver

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"forst/internal/compiler"
	"forst/internal/configiface"
	"forst/internal/discovery"
	"forst/internal/executor"

	logrus "github.com/sirupsen/logrus"
)

// HTTPContractVersion is the normative HTTP API revision (see examples/in/rfc/typescript-client/02-forst-dev-http-contract.md).
const HTTPContractVersion = "2"

// devFunctionExecutor is implemented by *executor.FunctionExecutor; HTTP tests may substitute stubs.
type devFunctionExecutor interface {
	ExecuteFunction(packageName, functionName string, args json.RawMessage) (*executor.ExecutionResult, error)
	ExecuteStreamingFunction(ctx context.Context, packageName, functionName string, args json.RawMessage) (<-chan executor.StreamingResult, error)
}

// devTypesGenerator is implemented by *TypeScriptGenerator; HTTP tests may substitute stubs.
type devTypesGenerator interface {
	GenerateTypesForFunctions(functions map[string]map[string]discovery.FunctionInfo, rootDir string) (string, error)
}

// InvokeRequest represents a request to call a Forst function.
type InvokeRequest struct {
	Package   string          `json:"package"`
	Function  string          `json:"function"`
	Args      json.RawMessage `json:"args"`
	Streaming bool            `json:"streaming,omitempty"`
}

// DevServerResponse represents a response to the client.
type DevServerResponse struct {
	Success bool            `json:"success"`
	Output  string          `json:"output,omitempty"`
	Error   string          `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

// BuildInfo is compiler build metadata served from GET /version.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// HTTPOpts holds HTTP server options (from config server section).
type HTTPOpts struct {
	ReadTimeoutSec  int
	WriteTimeoutSec int
	CORS            bool
}

// DevServer handles HTTP communication for Forst applications.
type DevServer struct {
	port       string
	server     *http.Server
	compiler   *compiler.Compiler
	log        *logrus.Logger
	config     configiface.ForstConfigIface
	httpOpts   HTTPOpts
	buildInfo  BuildInfo
	discoverer *discovery.Discoverer
	fnExec     devFunctionExecutor
	functions  map[string]map[string]discovery.FunctionInfo
	mu         sync.RWMutex
	// TypeScript generation cache
	typesCache     map[string]string // file path -> generated types
	typesCacheMu   sync.RWMutex
	lastTypesGen   time.Time
	typesGenerator devTypesGenerator
}

// NewHTTPServer creates a new HTTP server.
func NewHTTPServer(port string, comp *compiler.Compiler, log *logrus.Logger, cfg configiface.ForstConfigIface, buildInfo BuildInfo, httpOpts HTTPOpts, rootDir string) *DevServer {
	discoverer := discovery.NewDiscoverer(rootDir, log, cfg)
	fnExec := executor.NewFunctionExecutor(rootDir, comp, log, cfg)

	return &DevServer{
		port:       port,
		compiler:   comp,
		log:        log,
		config:     cfg,
		httpOpts:   httpOpts,
		buildInfo:  buildInfo,
		discoverer: discoverer,
		fnExec:     fnExec,
		functions:  make(map[string]map[string]discovery.FunctionInfo),
		typesCache: make(map[string]string),
	}
}
