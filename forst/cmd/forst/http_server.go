package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"forst/cmd/forst/compiler"
	"forst/internal/discovery"
	"forst/internal/executor"

	logrus "github.com/sirupsen/logrus"
)

// InvokeRequest represents a request to call a Forst function
type InvokeRequest struct {
	Package   string          `json:"package"`
	Function  string          `json:"function"`
	Args      json.RawMessage `json:"args"`
	Streaming bool            `json:"streaming,omitempty"`
}

// HTTPResponse represents a response to the client
type HTTPResponse struct {
	Success bool            `json:"success"`
	Output  string          `json:"output,omitempty"`
	Error   string          `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

// HTTPServer handles HTTP communication for Forst applications
type HTTPServer struct {
	port       string
	server     *http.Server
	compiler   *compiler.Compiler
	log        *logrus.Logger
	config     *ForstConfig
	discoverer *discovery.Discoverer
	executor   *executor.FunctionExecutor
	functions  map[string]map[string]discovery.FunctionInfo
	mu         sync.RWMutex
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(port string, comp *compiler.Compiler, log *logrus.Logger, config *ForstConfig, rootDir string) *HTTPServer {
	discoverer := discovery.NewDiscoverer(rootDir, log, config)
	executor := executor.NewFunctionExecutor(rootDir, comp, log, config)

	return &HTTPServer{
		port:       port,
		compiler:   comp,
		log:        log,
		config:     config,
		discoverer: discoverer,
		executor:   executor,
		functions:  make(map[string]map[string]discovery.FunctionInfo),
	}
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	// Discover functions on startup
	if err := s.discoverFunctions(); err != nil {
		s.log.Warnf("Failed to discover functions on startup: %v", err)
	}

	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealth)

	// Function discovery endpoint
	mux.HandleFunc("/functions", s.handleFunctions)

	// Function invocation endpoint
	mux.HandleFunc("/invoke", s.handleInvoke)

	s.server = &http.Server{
		Addr:         ":" + s.port,
		Handler:      mux,
		ReadTimeout:  time.Duration(s.config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.config.Server.WriteTimeout) * time.Second,
	}

	s.log.Infof("HTTP server listening on port %s", s.port)
	s.log.Info("Available endpoints:")
	s.log.Info("  GET  /health     - Health check")
	s.log.Info("  GET  /functions  - Discover available functions")
	s.log.Info("  POST /invoke     - Invoke a Forst function")

	return s.server.ListenAndServe()
}

// Stop stops the HTTP server
func (s *HTTPServer) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// discoverFunctions discovers all available functions
func (s *HTTPServer) discoverFunctions() error {
	functions, err := s.discoverer.DiscoverFunctions()
	if err != nil {
		return fmt.Errorf("failed to discover functions: %v", err)
	}

	s.mu.Lock()
	s.functions = functions
	s.mu.Unlock()

	s.log.Infof("Discovered %d packages with public functions", len(functions))
	return nil
}

// handleHealth handles health check requests
func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := HTTPResponse{
		Success: true,
		Output:  "Forst HTTP server is healthy",
	}

	s.sendJSONResponse(w, response)
}

// handleFunctions handles function discovery requests
func (s *HTTPServer) handleFunctions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Refresh function discovery
	if err := s.discoverFunctions(); err != nil {
		s.sendError(w, fmt.Sprintf("Failed to discover functions: %v", err), http.StatusInternalServerError)
		return
	}

	s.mu.RLock()
	functions := make([]discovery.FunctionInfo, 0)
	for _, pkgFuncs := range s.functions {
		for _, fn := range pkgFuncs {
			functions = append(functions, fn)
		}
	}
	s.mu.RUnlock()

	response := HTTPResponse{
		Success: true,
		Output:  fmt.Sprintf("Found %d public functions", len(functions)),
	}

	// Include function list in result
	if resultData, err := json.Marshal(functions); err == nil {
		response.Result = resultData
	}

	s.sendJSONResponse(w, response)
}

// handleInvoke handles function invocation requests
func (s *HTTPServer) handleInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req InvokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, fmt.Sprintf("Failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate function exists
	s.mu.RLock()
	pkgFuncs, ok := s.functions[req.Package]
	if !ok {
		s.mu.RUnlock()
		s.sendError(w, fmt.Sprintf("Package %s not found", req.Package), http.StatusNotFound)
		return
	}

	fn, ok := pkgFuncs[req.Function]
	s.mu.RUnlock()
	if !ok {
		s.sendError(w, fmt.Sprintf("Function %s not found in package %s", req.Function, req.Package), http.StatusNotFound)
		return
	}

	// Check streaming compatibility
	if req.Streaming && !fn.SupportsStreaming {
		s.sendError(w, fmt.Sprintf("Function %s does not support streaming", req.Function), http.StatusBadRequest)
		return
	}

	// Set up streaming if requested
	if req.Streaming {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Transfer-Encoding", "chunked")
		flusher, ok := w.(http.Flusher)
		if !ok {
			s.sendError(w, "Streaming not supported by server", http.StatusInternalServerError)
			return
		}

		// Execute streaming function
		results, err := s.executor.ExecuteStreamingFunction(r.Context(), req.Package, req.Function, req.Args)
		if err != nil {
			s.sendError(w, fmt.Sprintf("Streaming execution failed: %v", err), http.StatusInternalServerError)
			return
		}

		// Stream results back to client
		encoder := json.NewEncoder(w)
		for result := range results {
			if err := encoder.Encode(result); err != nil {
				s.log.Errorf("Failed to encode streaming result: %v", err)
				return
			}
			flusher.Flush()
		}
	} else {
		// Execute function normally
		result, err := s.executor.ExecuteFunction(req.Package, req.Function, req.Args)
		if err != nil {
			s.sendError(w, fmt.Sprintf("Function execution failed: %v", err), http.StatusInternalServerError)
			return
		}

		response := HTTPResponse{
			Success: result.Success,
			Output:  result.Output,
			Error:   result.Error,
			Result:  result.Result,
		}
		s.sendJSONResponse(w, response)
	}
}

// sendJSONResponse sends a JSON response to the client
func (s *HTTPServer) sendJSONResponse(w http.ResponseWriter, response HTTPResponse) {
	w.Header().Set("Content-Type", "application/json")

	if s.config.Server.CORS {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	}

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

// sendError sends an error response to the client
func (s *HTTPServer) sendError(w http.ResponseWriter, errorMsg string, statusCode int) {
	response := HTTPResponse{
		Success: false,
		Error:   errorMsg,
	}

	w.WriteHeader(statusCode)
	s.sendJSONResponse(w, response)
}

// StartDevServer is the entry point for the dev server command
func StartDevServer(port string, log *logrus.Logger, configPath string, rootDir string) {
	// Load configuration
	config, err := LoadConfig(configPath)
	if err != nil {
		log.Errorf("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	log.Infof("Loaded config: %+v", config)
	log.Infof("Config log level: %s", config.Dev.LogLevel)

	// Validate configuration
	if err := config.Validate(); err != nil {
		log.Errorf("Invalid configuration: %v", err)
		os.Exit(1)
	}

	// Override port if provided
	if port != "" {
		config.Server.Port = port
	}

	// Set log level based on config
	switch config.Dev.LogLevel {
	case "debug":
		log.SetLevel(logrus.DebugLevel)
		log.Debug("Log level set to DEBUG")
	case "info":
		log.SetLevel(logrus.InfoLevel)
		log.Debug("Log level set to INFO")
	case "warn":
		log.SetLevel(logrus.WarnLevel)
		log.Debug("Log level set to WARN")
	case "error":
		log.SetLevel(logrus.ErrorLevel)
		log.Debug("Log level set to ERROR")
	}

	// Create compiler with config
	args := config.ToCompilerArgs()
	comp := compiler.New(args, log)

	// Create server
	server := NewHTTPServer(config.Server.Port, comp, log, config, rootDir)

	log.Infof("Starting Forst dev server on port %s", config.Server.Port)
	log.Infof("Root directory: %s", rootDir)

	if err := server.Start(); err != nil {
		log.Errorf("HTTP server error: %v", err)
		os.Exit(1)
	}
}
