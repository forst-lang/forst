package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"forst/internal/compiler"
	"forst/internal/discovery"
	"forst/internal/executor"

	logrus "github.com/sirupsen/logrus"
)

// devFunctionExecutor is implemented by *executor.FunctionExecutor; HTTP tests may substitute stubs.
type devFunctionExecutor interface {
	ExecuteFunction(packageName, functionName string, args json.RawMessage) (*executor.ExecutionResult, error)
	ExecuteStreamingFunction(ctx context.Context, packageName, functionName string, args json.RawMessage) (<-chan executor.StreamingResult, error)
}

// devHTTPContractVersion is the normative HTTP API revision (see examples/in/rfc/typescript-client/02-forst-dev-http-contract.md).
const devHTTPContractVersion = "1"

// InvokeRequest represents a request to call a Forst function
type InvokeRequest struct {
	Package   string          `json:"package"`
	Function  string          `json:"function"`
	Args      json.RawMessage `json:"args"`
	Streaming bool            `json:"streaming,omitempty"`
}

// DevServerResponse represents a response to the client
type DevServerResponse struct {
	Success bool            `json:"success"`
	Output  string          `json:"output,omitempty"`
	Error   string          `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

// DevServer handles HTTP communication for Forst applications
type DevServer struct {
	port       string
	server     *http.Server
	compiler   *compiler.Compiler
	log        *logrus.Logger
	config     *ForstConfig
	discoverer *discovery.Discoverer
	fnExec     devFunctionExecutor
	functions  map[string]map[string]discovery.FunctionInfo
	mu         sync.RWMutex
	// TypeScript generation cache
	typesCache     map[string]string // file path -> generated types
	typesCacheMu   sync.RWMutex
	lastTypesGen   time.Time
	typesGenerator *TypeScriptGenerator
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(port string, comp *compiler.Compiler, log *logrus.Logger, config *ForstConfig, rootDir string) *DevServer {
	discoverer := discovery.NewDiscoverer(rootDir, log, config)
	fnExec := executor.NewFunctionExecutor(rootDir, comp, log, config)

	return &DevServer{
		port:       port,
		compiler:   comp,
		log:        log,
		config:     config,
		discoverer: discoverer,
		fnExec:     fnExec,
		functions:  make(map[string]map[string]discovery.FunctionInfo),
		typesCache: make(map[string]string),
	}
}
