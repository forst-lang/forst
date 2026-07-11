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
	"forst/internal/invokeserver"

	logrus "github.com/sirupsen/logrus"
)

// devHTTPContractVersion is the normative HTTP API revision (see examples/in/rfc/typescript-client/02-forst-dev-http-contract.md).
const devHTTPContractVersion = invokeserver.HTTPContractVersion

// InvokeRequest represents a request to call a Forst function.
type InvokeRequest = invokeserver.InvokeRequest

// DevServerResponse represents a response to the client.
type DevServerResponse = invokeserver.Response

// devFunctionExecutor is implemented by *executor.FunctionExecutor; HTTP tests may substitute stubs.
type devFunctionExecutor interface {
	ExecuteFunction(packageName, functionName string, args json.RawMessage) (*executor.ExecutionResult, error)
	ExecuteStreamingFunction(ctx context.Context, packageName, functionName string, args json.RawMessage) (<-chan executor.StreamingResult, error)
}

// devTypesGenerator is implemented by *TypeScriptGenerator; HTTP tests may substitute stubs.
type devTypesGenerator interface {
	GenerateTypesForFunctions(functions map[string]map[string]discovery.FunctionInfo, rootDir string) (string, error)
}

// DevServer handles HTTP communication for Forst applications.
type DevServer struct {
	invoke         *invokeserver.Server
	devBackend     *invokeserver.DevBackend
	port           string
	server         *http.Server
	compiler       *compiler.Compiler
	log            *logrus.Logger
	config         *ForstConfig
	discoverer     *discovery.Discoverer
	fnExec         devFunctionExecutor
	functions      map[string]map[string]discovery.FunctionInfo
	mu             sync.RWMutex
	typesCache     map[string]string
	typesCacheMu   sync.RWMutex
	lastTypesGen   time.Time
	typesGenerator devTypesGenerator
}

// NewHTTPServer creates a new HTTP server.
func NewHTTPServer(port string, comp *compiler.Compiler, log *logrus.Logger, config *ForstConfig, rootDir string) *DevServer {
	discoverer := discovery.NewDiscoverer(rootDir, log, config)
	fnExec := executor.NewFunctionExecutor(rootDir, comp, log, config)
	backend := invokeserver.NewDevBackend(fnExec, discoverer)
	invokeCfg := invokeserver.Config{
		Host:           config.Server.Host,
		Port:           port,
		CORS:           config.Server.CORS,
		ReadTimeout:    config.Server.ReadTimeout,
		WriteTimeout:   config.Server.WriteTimeout,
		MaxRequestSize: config.Server.MaxRequestSize,
		Runtime:        "dev",
	}
	version := invokeserver.VersionInfo{
		Version:         Version,
		Commit:          Commit,
		Date:            Date,
		ContractVersion: devHTTPContractVersion,
		Runtime:         "dev",
	}
	invokeServer := invokeserver.New(invokeCfg, backend, version, log)

	return &DevServer{
		port:       port,
		invoke:     invokeServer,
		devBackend: backend,
		fnExec:     fnExec,
		compiler:   comp,
		log:        log,
		config:     config,
		discoverer: discoverer,
		functions:  make(map[string]map[string]discovery.FunctionInfo),
		typesCache: make(map[string]string),
	}
}

// setInvokeBackendForTest swaps the invoke dispatch backend (tests only).
func (s *DevServer) setInvokeBackendForTest(backend invokeserver.DispatchBackend) {
	if s.invoke != nil {
		s.invoke.SetBackend(backend)
	}
}

// syncFunctionsCache mirrors backend discovery into the legacy functions field for tests.
func (s *DevServer) syncFunctionsCache() {
	if s.invoke == nil {
		return
	}
	s.mu.Lock()
	s.functions = s.invoke.BackendFunctions()
	s.mu.Unlock()
}
