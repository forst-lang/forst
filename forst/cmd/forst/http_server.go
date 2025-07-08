package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"forst/cmd/forst/compiler"

	logrus "github.com/sirupsen/logrus"
)

// HTTPRequest represents a request from the Node.js client
type HTTPRequest struct {
	Action   string          `json:"action"`
	TestFile string          `json:"testFile,omitempty"`
	Args     json.RawMessage `json:"args,omitempty"`
}

// HTTPResponse represents a response to the Node.js client
type HTTPResponse struct {
	Success bool            `json:"success"`
	Output  string          `json:"output,omitempty"`
	Error   string          `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

// HTTPServer handles HTTP communication for Forst applications
type HTTPServer struct {
	port     string
	server   *http.Server
	compiler *compiler.Compiler
	log      *logrus.Logger
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(port string, comp *compiler.Compiler, log *logrus.Logger) *HTTPServer {
	return &HTTPServer{
		port:     port,
		compiler: comp,
		log:      log,
	}
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealth)
	
	// Run test endpoint
	mux.HandleFunc("/run", s.handleRun)
	
	// Compile endpoint
	mux.HandleFunc("/compile", s.handleCompile)

	s.server = &http.Server{
		Addr:         ":" + s.port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	s.log.Infof("HTTP server listening on port %s", s.port)
	return s.server.ListenAndServe()
}

// Stop stops the HTTP server
func (s *HTTPServer) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
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

// handleRun handles run requests
func (s *HTTPServer) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request HTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.sendError(w, fmt.Sprintf("Failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	if request.TestFile == "" {
		s.sendError(w, "Test file is required", http.StatusBadRequest)
		return
	}

	// Run the test file and capture output
	output, errorOutput, err := s.runTestFile(request.TestFile)
	if err != nil {
		s.sendError(w, fmt.Sprintf("Failed to run test: %v", err), http.StatusInternalServerError)
		return
	}

	response := HTTPResponse{
		Success: true,
		Output:  output,
		Error:   errorOutput,
	}

	s.sendJSONResponse(w, response)
}

// handleCompile handles compile requests
func (s *HTTPServer) handleCompile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request HTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.sendError(w, fmt.Sprintf("Failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	if request.TestFile == "" {
		s.sendError(w, "Test file is required", http.StatusBadRequest)
		return
	}

	// Compile the test file
	err := s.compileFile(request.TestFile)
	if err != nil {
		s.sendError(w, fmt.Sprintf("Compilation failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := HTTPResponse{
		Success: true,
		Output:  "Compilation successful",
	}

	s.sendJSONResponse(w, response)
}

// compileFile compiles a Forst file
func (s *HTTPServer) compileFile(testFile string) error {
	// Create a new compiler instance for this file
	args := compiler.Args{
		FilePath: testFile,
	}
	
	comp := compiler.New(args, s.log)
	
	// Compile the file
	_, err := comp.CompileFile()
	return err
}

// runTestFile runs a test file and captures output
func (s *HTTPServer) runTestFile(testFile string) (string, string, error) {
	// First compile the file
	if err := s.compileFile(testFile); err != nil {
		return "", "", fmt.Errorf("compilation failed: %v", err)
	}

	// Then run it using the existing compiler infrastructure
	return s.executeTestFile(testFile)
}

// executeTestFile executes a test file and captures its output
func (s *HTTPServer) executeTestFile(testFile string) (string, string, error) {
	// Create a temporary output file
	args := compiler.Args{
		FilePath: testFile,
	}
	
	comp := compiler.New(args, s.log)
	
	// Compile to get Go code
	goCode, err := comp.CompileFile()
	if err != nil {
		return "", "", fmt.Errorf("compilation failed: %v", err)
	}

	// Create temporary file
	outputPath, err := compiler.CreateTempOutputFile(*goCode)
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.RemoveAll(filepath.Dir(outputPath))

	// Run the compiled Go code
	cmd := exec.Command("go", "run", outputPath)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return "", string(output), err
	}

	return string(output), "", nil
}

// sendJSONResponse sends a JSON response to the client
func (s *HTTPServer) sendJSONResponse(w http.ResponseWriter, response HTTPResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

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
func StartDevServer(port string, log *logrus.Logger) {
	args := compiler.Args{}
	comp := compiler.New(args, log)
	server := NewHTTPServer(port, comp, log)

	log.Infof("Starting Forst dev server on port %s", port)
	log.Info("Available endpoints:")
	log.Info("  GET  /health   - Health check")
	log.Info("  POST /run       - Run a Forst test file")
	log.Info("  POST /compile   - Compile a Forst file")

	if err := server.Start(); err != nil {
		log.Errorf("HTTP server error: %v", err)
		os.Exit(1)
	}
} 