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
	transformerts "forst/internal/transformer/ts"

	logrus "github.com/sirupsen/logrus"
)

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
	executor   *executor.FunctionExecutor
	functions  map[string]map[string]discovery.FunctionInfo
	mu         sync.RWMutex
	// TypeScript generation cache
	typesCache     map[string]string // file path -> generated types
	typesCacheMu   sync.RWMutex
	lastTypesGen   time.Time
	typesGenerator *TypeScriptGenerator
}

// TypeScriptGenerator handles TypeScript type generation
type TypeScriptGenerator struct {
	log *logrus.Logger
}

// NewTypeScriptGenerator creates a new TypeScript generator
func NewTypeScriptGenerator(log *logrus.Logger) *TypeScriptGenerator {
	return &TypeScriptGenerator{
		log: log,
	}
}

// GenerateTypesForFunctions generates TypeScript types for discovered functions
func (tg *TypeScriptGenerator) GenerateTypesForFunctions(functions map[string]map[string]discovery.FunctionInfo, rootDir string) (string, error) {
	// Collect all Forst files that contain discovered functions
	filePaths := make(map[string]bool)
	for _, pkgFuncs := range functions {
		for _, fn := range pkgFuncs {
			filePaths[fn.FilePath] = true
		}
	}

	var outputs []*transformerts.TypeScriptOutput
	for filePath := range filePaths {
		out, err := transformerts.TransformForstFileFromPath(filePath, tg.log, transformerts.TransformForstFileOptions{
			RelaxedTypecheck: true,
		})
		if err != nil {
			tg.log.Warnf("Failed to generate types for %s: %v", filePath, err)
			continue
		}
		outputs = append(outputs, out)
	}

	if len(outputs) == 0 {
		return (&transformerts.TypeScriptOutput{}).GenerateTypesFile(), nil
	}

	merged, err := transformerts.MergeTypeScriptOutputs(outputs)
	if err != nil {
		return "", err
	}

	return merged.GenerateTypesFile(), nil
}

// generateTypesForFile generates TypeScript types for a single Forst file
func (tg *TypeScriptGenerator) generateTypesForFile(filePath string) ([]string, []transformerts.FunctionSignature, string, error) {
	out, err := transformerts.TransformForstFileFromPath(filePath, tg.log, transformerts.TransformForstFileOptions{
		RelaxedTypecheck: true,
	})
	if err != nil {
		return nil, nil, "", err
	}
	return out.Types, out.Functions, out.PackageName, nil
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(port string, comp *compiler.Compiler, log *logrus.Logger, config *ForstConfig, rootDir string) *DevServer {
	discoverer := discovery.NewDiscoverer(rootDir, log, config)
	executor := executor.NewFunctionExecutor(rootDir, comp, log, config)

	return &DevServer{
		port:       port,
		compiler:   comp,
		log:        log,
		config:     config,
		discoverer: discoverer,
		executor:   executor,
		functions:  make(map[string]map[string]discovery.FunctionInfo),
		typesCache: make(map[string]string),
	}
}

// Start starts the HTTP server
func (s *DevServer) Start() error {
	if err := s.discoverFunctions(); err != nil {
		s.log.Warnf("Failed to discover functions on startup: %v", err)
	}

	// Initialize TypeScript generator
	s.typesGenerator = NewTypeScriptGenerator(s.log)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/functions", s.handleFunctions)
	mux.HandleFunc("/invoke", s.handleInvoke)
	mux.HandleFunc("/types", s.handleTypes) // New endpoint for TypeScript types

	s.server = &http.Server{
		Addr:         ":" + s.port,
		Handler:      mux,
		ReadTimeout:  time.Duration(s.config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.config.Server.WriteTimeout) * time.Second,
	}

	s.logStartupInfo()

	return s.server.ListenAndServe()
}

// logStartupInfo logs information about the server startup
func (s *DevServer) logStartupInfo() {
	s.log.Infof("HTTP server listening on port %s", s.port)
	s.log.Info("Available endpoints:")
	s.log.Info("  GET  /functions  - Discover available functions")
	s.log.Info("  POST /invoke     - Invoke a Forst function")
	s.log.Info("  GET  /types      - Generate TypeScript types for discovered functions")
	s.log.Info("  GET  /health     - Health check")
}

// Stop stops the HTTP server
func (s *DevServer) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// discoverFunctions discovers all available functions
func (s *DevServer) discoverFunctions() error {
	functions, err := s.discoverer.DiscoverFunctions()
	if err != nil {
		return fmt.Errorf("failed to discover functions: %v", err)
	}

	s.mu.Lock()
	s.functions = functions
	s.mu.Unlock()

	return nil
}

// handleHealth handles health check requests
func (s *DevServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := DevServerResponse{
		Success: true,
		Output:  "Forst HTTP server is healthy",
	}

	s.sendJSONResponse(w, response)
}

// handleFunctions handles function discovery requests
func (s *DevServer) handleFunctions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	response := DevServerResponse{
		Success: true,
	}

	// Include function list in result
	if resultData, err := json.Marshal(functions); err == nil {
		response.Result = resultData
	}

	s.sendJSONResponse(w, response)
}

// handleInvoke handles function invocation requests
func (s *DevServer) handleInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
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

		response := DevServerResponse{
			Success: result.Success,
			Output:  result.Output,
			Error:   result.Error,
			Result:  result.Result,
		}
		s.sendJSONResponse(w, response)
	}
}

// handleTypes handles TypeScript type generation requests
func (s *DevServer) handleTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if we need to regenerate types (cache invalidation)
	forceRegenerate := r.URL.Query().Get("force") == "true"

	s.typesCacheMu.RLock()
	_, exists := s.typesCache["types"]
	lastGen := s.lastTypesGen
	s.typesCacheMu.RUnlock()

	// Regenerate if forced, cache doesn't exist, or it's been more than 5 minutes
	shouldRegenerate := forceRegenerate || !exists || time.Since(lastGen) > 5*time.Minute

	if shouldRegenerate {
		s.log.Debug("Regenerating TypeScript types...")

		// Refresh function discovery
		if err := s.discoverFunctions(); err != nil {
			s.sendError(w, fmt.Sprintf("Failed to discover functions: %v", err), http.StatusInternalServerError)
			return
		}

		s.mu.RLock()
		functions := s.functions
		s.mu.RUnlock()

		// Generate TypeScript types
		typesContent, err := s.typesGenerator.GenerateTypesForFunctions(functions, s.discoverer.GetRootDir())
		if err != nil {
			s.sendError(w, fmt.Sprintf("Failed to generate TypeScript types: %v", err), http.StatusInternalServerError)
			return
		}

		// Cache the generated types
		s.typesCacheMu.Lock()
		s.typesCache["types"] = typesContent
		s.lastTypesGen = time.Now()
		s.typesCacheMu.Unlock()

		s.log.Debug("TypeScript types generated and cached")
	} else {
		s.log.Debug("Using cached TypeScript types")
	}

	// Return the types
	s.typesCacheMu.RLock()
	typesContent := s.typesCache["types"]
	s.typesCacheMu.RUnlock()

	// JSON envelope with merged TS in `output` (Content-Type from sendJSONResponse).
	response := DevServerResponse{
		Success: true,
		Output:  typesContent,
	}

	s.sendJSONResponse(w, response)
}

// sendJSONResponse sends a JSON response to the client
func (s *DevServer) sendJSONResponse(w http.ResponseWriter, response DevServerResponse) {
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
func (s *DevServer) sendError(w http.ResponseWriter, errorMsg string, statusCode int) {
	response := DevServerResponse{
		Success: false,
		Error:   errorMsg,
	}

	w.WriteHeader(statusCode)
	s.sendJSONResponse(w, response)
}

// StartDevServer is the entry point for the dev server command
func StartDevServer(port string, log *logrus.Logger, configPath string, rootDir string, logLevel *string) {
	config := loadAndValidateConfig(configPath, log, port, logLevel)

	// Create compiler with config
	args := config.ToCompilerArgs()
	comp := compiler.New(args, log)

	// Create server
	server := NewHTTPServer(config.Server.Port, comp, log, config, rootDir)

	log.Debugf("Starting Forst dev server on port %s", config.Server.Port)
	log.Debugf("Root directory: %s", rootDir)

	if err := server.Start(); err != nil {
		log.Errorf("HTTP server error: %v", err)
		os.Exit(1)
	}
}

func loadAndValidateConfig(configPath string, log *logrus.Logger, port string, logLevel *string) *ForstConfig {
	config, err := LoadConfig(configPath)
	if err != nil {
		log.Errorf("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	if configPath == "" {
		log.Infof("No config file provided, using default configuration")
	} else {
		log.Infof("Loaded config from: %s", configPath)
	}

	// Override port if provided
	if port != "" && port != config.Server.Port {
		config.Server.Port = port
	}

	// Override log level if provided
	if logLevel != nil {
		config.Dev.LogLevel = *logLevel
	}

	type configSection struct {
		name    string
		entries []string
	}
	sections := []configSection{
		{"Server", []string{
			fmt.Sprintf("%-15s %s", "Port:", config.Server.Port),
			fmt.Sprintf("%-15s %v", "CORS enabled:", config.Server.CORS),
		}},
		{"Compiler", []string{
			fmt.Sprintf("%-15s %s", "Target:", config.Compiler.Target),
			fmt.Sprintf("%-15s %s", "Optimization:", config.Compiler.Optimization),
			fmt.Sprintf("%-15s %v", "Report phases:", config.Compiler.ReportPhases),
		}},
		{"Files", []string{
			fmt.Sprintf("%-15s %s", "Include:", config.Files.Include),
			fmt.Sprintf("%-15s %s", "Exclude:", config.Files.Exclude),
		}},
		{"Output", []string{
			fmt.Sprintf("%-15s %s", "Dir:", config.Output.Dir),
			fmt.Sprintf("%-15s %s", "File name:", config.Output.FileName),
			fmt.Sprintf("%-15s %v", "Source maps:", config.Output.SourceMaps),
			fmt.Sprintf("%-15s %v", "Clean:", config.Output.Clean),
		}},
		{"Dev", []string{
			fmt.Sprintf("%-15s %v", "Hot reload:", config.Dev.HotReload),
			fmt.Sprintf("%-15s %v", "Watch:", config.Dev.Watch),
			fmt.Sprintf("%-15s %v", "Auto restart:", config.Dev.AutoRestart),
			fmt.Sprintf("%-15s %s", "Log level:", config.Dev.LogLevel),
			fmt.Sprintf("%-15s %v", "Verbose:", config.Dev.Verbose),
		}},
	}

	// Set log level based on config
	switch config.Dev.LogLevel {
	case "trace":
		log.SetLevel(logrus.TraceLevel)
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "info":
		log.SetLevel(logrus.InfoLevel)
	case "warn":
		log.SetLevel(logrus.WarnLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	}

	for _, section := range sections {
		log.Debugf("%s:", section.name)
		for _, entry := range section.entries {
			log.Debugf("    %s", entry)
		}
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		log.Errorf("Invalid configuration: %v", err)
		os.Exit(1)
	}

	return config
}
