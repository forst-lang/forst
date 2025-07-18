package lsp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/parser"
	transformer_go "forst/internal/transformer/go"
	"forst/internal/typechecker"

	"runtime"

	"github.com/sirupsen/logrus"
)

// LSPRequest represents an LSP request
type LSPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// LSPResponse represents an LSP response
type LSPServerResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *LSPError   `json:"error,omitempty"`
}

// LSPError represents an LSP error
type LSPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// LSPServer represents the LSP server
type LSPServer struct {
	debugger    CompilerDebuggerInterface
	lspDebugger *LSPDebugger
	log         *logrus.Logger
	port        string
	server      *http.Server
}

// Version information for LSP server
var (
	// Version is the current version of Forst
	Version = "dev"
	// Commit is the git commit hash
	Commit = "unknown"
	// Date is the build date
	Date = "unknown"
)

// NewLSPServer creates a new LSP server
func NewLSPServer(port string, log *logrus.Logger) *LSPServer {
	debugger := NewCompilerDebugger(true)
	lspDebugger := NewLSPDebugger(debugger, "")

	return &LSPServer{
		debugger:    debugger,
		lspDebugger: lspDebugger,
		log:         log,
		port:        port,
	}
}

// Start starts the LSP server
func (s *LSPServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleLSP)
	mux.HandleFunc("/health", s.handleHealth)

	s.server = &http.Server{
		Addr:         ":" + s.port,
		Handler:      s.recoveryMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	s.log.Infof("LSP server listening on port %s", s.port)
	s.log.Info("Available endpoints:")
	s.log.Info("  POST / - LSP protocol endpoint")
	s.log.Info("  GET  /health - Health check")

	return s.server.ListenAndServe()
}

// recoveryMiddleware adds panic recovery to HTTP requests
func (s *LSPServer) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				s.log.Errorf("Panic in HTTP handler: %v", r)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// handleHealth handles health check requests
func (s *LSPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "forst-lsp",
		"version":   Version,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// Stop stops the LSP server
func (s *LSPServer) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// handleLSP handles LSP protocol requests
func (s *LSPServer) handleLSP(w http.ResponseWriter, r *http.Request) {
	// Add panic recovery to prevent server crashes
	defer func() {
		if r := recover(); r != nil {
			s.log.Errorf("Panic in LSP handler: %v", r)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}()

	// Only handle POST requests for LSP protocol
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.log.Errorf("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}

	// Parse LSP request
	var request LSPRequest
	if err := json.Unmarshal(body, &request); err != nil {
		s.log.Errorf("Failed to parse LSP request: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Handle different LSP methods
	response := s.handleLSPMethod(request)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	responseJSON, err := json.Marshal(response)
	if err != nil {
		s.log.Errorf("Failed to marshal response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Write(responseJSON)
}

// handleLSPMethod handles different LSP methods
func (s *LSPServer) handleLSPMethod(request LSPRequest) LSPServerResponse {
	switch request.Method {
	case "initialize":
		return s.handleInitialize(request)
	case "textDocument/didOpen":
		return s.handleDidOpen(request)
	case "textDocument/didChange":
		return s.handleDidChange(request)
	case "textDocument/didClose":
		return s.handleDidClose(request)
	case "textDocument/publishDiagnostics":
		return s.handlePublishDiagnostics(request)
	case "textDocument/hover":
		return s.handleHover(request)
	case "textDocument/completion":
		return s.handleCompletion(request)
	case "shutdown":
		return s.handleShutdown(request)
	case "exit":
		return s.handleExit(request)
	default:
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &LSPError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", request.Method),
			},
		}
	}
}

// handleInitialize handles the initialize method
func (s *LSPServer) handleInitialize(request LSPRequest) LSPServerResponse {
	capabilities := map[string]interface{}{
		"textDocumentSync": map[string]interface{}{
			"openClose": true,
			"change":    1, // Incremental
		},
		"completionProvider": map[string]interface{}{
			"triggerCharacters": []string{".", ":", "("},
		},
		"hoverProvider": true,
		"diagnosticProvider": map[string]interface{}{
			"identifier": "forst",
		},
	}

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: map[string]interface{}{
			"capabilities": capabilities,
			"serverInfo": map[string]interface{}{
				"name":    "forst-lsp",
				"version": Version,
			},
		},
	}
}

// handleDidOpen handles the textDocument/didOpen method
func (s *LSPServer) handleDidOpen(request LSPRequest) LSPServerResponse {
	// Add panic recovery to prevent server crashes
	defer func() {
		if r := recover(); r != nil {
			s.log.Errorf("Panic in handleDidOpen: %v", r)
		}
	}()

	// Parse the document URI and content
	var params struct {
		TextDocument struct {
			URI     string `json:"uri"`
			Version int    `json:"version"`
			Text    string `json:"text"`
		} `json:"textDocument"`
	}

	if err := json.Unmarshal(request.Params, &params); err != nil {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &LSPError{
				Code:    -32700,
				Message: "Parse error",
			},
		}
	}

	// Process the Forst file and generate diagnostics
	diagnostics := s.processForstFile(params.TextDocument.URI, params.TextDocument.Text)

	// Send diagnostics notification
	s.sendDiagnosticsNotification(params.TextDocument.URI, diagnostics)

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  nil,
	}
}

// handleDidChange handles the textDocument/didChange method
func (s *LSPServer) handleDidChange(request LSPRequest) LSPServerResponse {
	// Parse the document changes
	var params struct {
		TextDocument struct {
			URI     string `json:"uri"`
			Version int    `json:"version"`
		} `json:"textDocument"`
		ContentChanges []struct {
			Text string `json:"text"`
		} `json:"contentChanges"`
	}

	if err := json.Unmarshal(request.Params, &params); err != nil {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &LSPError{
				Code:    -32700,
				Message: "Parse error",
			},
		}
	}

	// Process the updated content
	if len(params.ContentChanges) > 0 {
		diagnostics := s.processForstFile(params.TextDocument.URI, params.ContentChanges[0].Text)
		s.sendDiagnosticsNotification(params.TextDocument.URI, diagnostics)
	}

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  nil,
	}
}

// handleDidClose handles the textDocument/didClose method
func (s *LSPServer) handleDidClose(request LSPRequest) LSPServerResponse {
	// Clear diagnostics for the closed document
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}

	if err := json.Unmarshal(request.Params, &params); err != nil {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &LSPError{
				Code:    -32700,
				Message: "Parse error",
			},
		}
	}

	// Send empty diagnostics to clear them
	s.sendDiagnosticsNotification(params.TextDocument.URI, []LSPDiagnostic{})

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  nil,
	}
}

// handlePublishDiagnostics handles the textDocument/publishDiagnostics method
func (s *LSPServer) handlePublishDiagnostics(request LSPRequest) LSPServerResponse {
	// This is typically a notification from the client, not a request
	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  nil,
	}
}

// handleHover handles the textDocument/hover method
func (s *LSPServer) handleHover(request LSPRequest) LSPServerResponse {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Position LSPPosition `json:"position"`
	}

	if err := json.Unmarshal(request.Params, &params); err != nil {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &LSPError{
				Code:    -32700,
				Message: "Parse error",
			},
		}
	}

	// Find hover information for the position
	hover := s.findHoverForPosition(params.TextDocument.URI, params.Position)

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  hover,
	}
}

// handleCompletion handles the textDocument/completion method
func (s *LSPServer) handleCompletion(request LSPRequest) LSPServerResponse {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Position LSPPosition `json:"position"`
	}

	if err := json.Unmarshal(request.Params, &params); err != nil {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &LSPError{
				Code:    -32700,
				Message: "Parse error",
			},
		}
	}

	// Get completion items for the position
	completions := s.getCompletionsForPosition(params.TextDocument.URI, params.Position)

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: map[string]interface{}{
			"isIncomplete": false,
			"items":        completions,
		},
	}
}

// handleShutdown handles the shutdown method
func (s *LSPServer) handleShutdown(request LSPRequest) LSPServerResponse {
	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  nil,
	}
}

// handleExit handles the exit method
func (s *LSPServer) handleExit(request LSPRequest) LSPServerResponse {
	// Exit the server
	go func() {
		time.Sleep(100 * time.Millisecond)
		os.Exit(0)
	}()

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  nil,
	}
}

// processForstFile processes a Forst file and returns diagnostics
func (s *LSPServer) processForstFile(uri, content string) []LSPDiagnostic {
	// Convert URI to file path
	filePath := strings.TrimPrefix(uri, "file://")
	if runtime.GOOS == "windows" {
		filePath = strings.TrimPrefix(filePath, "/")
	}

	// Only process .ft files
	if !strings.HasSuffix(filePath, ".ft") {
		return nil
	}

	// Create debugger for this file
	debugger := s.debugger.GetDebugger(PhaseTransformer, filePath)
	if debugger == nil {
		return nil
	}

	// Process the file through the compiler pipeline
	diagnostics := s.compileForstFile(filePath, content, debugger)

	return diagnostics
}

// compileForstFile compiles a Forst file and returns diagnostics
func (s *LSPServer) compileForstFile(filePath, content string, debugger Debugger) []LSPDiagnostic {
	// Add panic recovery to prevent LSP server crashes
	defer func() {
		if r := recover(); r != nil {
			s.log.Errorf("Panic in compileForstFile for %s: %v", filePath, r)
		}
	}()

	// Lexical analysis
	lex := lexer.New([]byte(content), filePath, s.log)
	tokens := lex.Lex()

	debugger.LogEvent(EventLexerComplete, "Lexical analysis completed", map[string]interface{}{
		"token_count": len(tokens),
		"file":        filePath,
	})

	// Parsing with panic recovery
	psr := parser.New(tokens, filePath, s.log)
	var astNodes []ast.Node
	var err error

	func() {
		defer func() {
			if r := recover(); r != nil {
				s.log.Errorf("Parser panic for %s: %v", filePath, r)
				err = fmt.Errorf("parser panic: %v", r)
			}
		}()
		astNodes, err = psr.ParseFile()
	}()

	if err != nil {
		// Create diagnostic for parsing error
		diagnostic := CreateTypeErrorDiagnostic(
			"file://"+filePath,
			1, // Default to line 1
			"",
			"",
			"parsing error",
		)
		diagnostic.Message = fmt.Sprintf("Parsing error: %v", err)
		return []LSPDiagnostic{diagnostic}
	}

	debugger.LogEvent(EventParserComplete, "Parsing completed", map[string]interface{}{
		"node_count": len(astNodes),
		"file":       filePath,
	})

	// Type checking
	tc := typechecker.New(s.log, false)
	if err := tc.CheckTypes(astNodes); err != nil {
		// Create diagnostic for type checking error
		diagnostic := CreateTypeErrorDiagnostic(
			"file://"+filePath,
			1, // Default to line 1
			"",
			"",
			"type checking error",
		)
		diagnostic.Message = fmt.Sprintf("Type checking error: %v", err)
		return []LSPDiagnostic{diagnostic}
	}

	debugger.LogEvent(EventTypecheckerComplete, "Type checking completed", map[string]interface{}{
		"file": filePath,
	})

	// Code transformation
	transformer := transformer_go.New(tc, s.log, false)
	_, err = transformer.TransformForstFileToGo(astNodes)
	if err != nil {
		// Create diagnostic for transformation error
		diagnostic := CreateTypeErrorDiagnostic(
			"file://"+filePath,
			1, // Default to line 1
			"",
			"",
			"transformation error",
		)
		diagnostic.Message = fmt.Sprintf("Transformation error: %v", err)
		return []LSPDiagnostic{diagnostic}
	}

	debugger.LogEvent(EventTransformerComplete, "Code transformation completed", map[string]interface{}{
		"file": filePath,
	})

	// Process debug events and convert to LSP diagnostics
	s.lspDebugger.ProcessDebugEvents()
	return s.lspDebugger.GetDiagnostics()
}

// findHoverForPosition finds hover information for a specific position
func (s *LSPServer) findHoverForPosition(uri string, position LSPPosition) *LSPHover {
	// For now, return a simple hover with file information
	filePath := strings.TrimPrefix(uri, "file://")
	if runtime.GOOS == "windows" {
		filePath = strings.TrimPrefix(filePath, "/")
	}

	content := fmt.Sprintf("**Forst File**\n\n**File:** %s\n**Line:** %d\n**Character:** %d",
		filepath.Base(filePath), position.Line+1, position.Character)

	return &LSPHover{
		Contents: LSPMarkedString{
			Language: "markdown",
			Value:    content,
		},
	}
}

// getCompletionsForPosition gets completion items for a specific position
func (s *LSPServer) getCompletionsForPosition(uri string, position LSPPosition) []LSPCompletionItem {
	var completions []LSPCompletionItem
	for keyword := range lexer.Keywords {
		completions = append(completions, LSPCompletionItem{
			Label:            keyword,
			Kind:             LSPCompletionItemKindKeyword,
			Detail:           fmt.Sprintf("Forst keyword: %s", keyword),
			Documentation:    fmt.Sprintf("Forst language keyword: %s", keyword),
			InsertText:       keyword,
			InsertTextFormat: LSPInsertTextFormatPlainText,
		})
	}

	return completions
}

// sendDiagnosticsNotification sends a diagnostics notification
func (s *LSPServer) sendDiagnosticsNotification(uri string, diagnostics []LSPDiagnostic) {
	// In a real LSP implementation, this would be sent to the client
	// For now, we just log it
	s.log.Debugf("Sending diagnostics for %s: %d diagnostics", uri, len(diagnostics))
}
