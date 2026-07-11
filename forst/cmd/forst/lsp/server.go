package lsp

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"forst/internal/ast"
	"forst/internal/httpbody"
	transformer_go "forst/internal/transformer/go"

	"github.com/sirupsen/logrus"
)

// LSPRequest represents an LSP request
type LSPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitzero"`
}

// LSPResponse represents an LSP response
type LSPServerResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Result  any       `json:"result,omitzero"`
	Error   *LSPError `json:"error,omitzero"`
}

// LSPError represents an LSP error
type LSPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitzero"`
}

// LSPServer represents the LSP server
type LSPServer struct {
	debugger    CompilerDebuggerInterface
	lspDebugger *LSPDebugger
	log         *logrus.Logger
	port        string
	server      *http.Server
	// Add debugging state
	debugMode   bool
	debugEvents []DebugEvent
	// openDocuments holds the latest buffer text. Keys are canonical file:// URIs (see canonicalFileURI);
	// the client’s raw URI may also be stored as an alias when it differs, so lookups stay consistent.
	documentMu    sync.RWMutex
	openDocuments map[string]string

	// peerAnalysisCache stores last *forstDocumentContext per canonical URI keyed by buffer text, used only for
	// cross-buffer completion (read-only symbol lists). Current-buffer analysis always calls analyzeForstDocument
	// so TypeChecker scope state is never reused after RestoreScope.
	peerAnalysisMu    sync.Mutex
	peerAnalysisCache map[string]peerAnalysisCacheEntry

	// packageAnalysis caches merged same-package snapshots by content fingerprint (bounded LRU).
	packageAnalysis *packageAnalysisLRU

	// maxRequestBodyBytes caps POST body size; 0 uses httpbody.DefaultMaxBytes.
	maxRequestBodyBytes int64
}

// peerAnalysisCacheEntry is a content-keyed snapshot for same-package peer buffers (completion only).
type peerAnalysisCacheEntry struct {
	content string
	ctx     *forstDocumentContext
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
		debugger:          debugger,
		lspDebugger:       lspDebugger,
		log:               log,
		port:              port,
		debugMode:         true, // Enable debug mode by default for LLM debugging
		debugEvents:       make([]DebugEvent, 0),
		openDocuments:     make(map[string]string),
		peerAnalysisCache: make(map[string]peerAnalysisCacheEntry),
		packageAnalysis:   newPackageAnalysisLRU(defaultPackageAnalysisCacheMaxEntries),
	}
}

// Start starts the LSP server
func (s *LSPServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleLSP)
	mux.HandleFunc("/health", s.handleHealth)

	s.server = &http.Server{
		Handler:      s.recoveryMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	ln, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		return err
	}
	s.log.Infof("LSP server listening on %s", ln.Addr().String())
	s.log.Info("Available endpoints:")
	s.log.Info("  POST / - LSP protocol endpoint")
	s.log.Info("  GET  /health - Health check")

	return s.server.Serve(ln)
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

// Stop stops the LSP server
func (s *LSPServer) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// handleLSP handles LSP protocol over HTTP: POST a single JSON-RPC 2.0 object per request.
// GET returns a small JSON hint for humans or probes (editors must use POST for RPC).
func (s *LSPServer) handleLSP(w http.ResponseWriter, r *http.Request) {
	// Add panic recovery to prevent server crashes
	defer func() {
		if r := recover(); r != nil {
			s.log.Errorf("Panic in LSP handler: %v", r)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}()

	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"service":"forst-lsp","detail":"POST a JSON-RPC 2.0 object (Content-Type: application/json) to this URL"}`))
		return
	}

	// LSP traffic uses POST with a JSON body
	if r.Method != http.MethodPost {
		s.log.Warnf("Invalid method %s for LSP endpoint", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	maxBody := s.maxRequestBodyBytes
	if maxBody <= 0 {
		maxBody = httpbody.DefaultMaxBytes
	}
	body, err := httpbody.ReadAll(r.Body, maxBody)
	if err != nil {
		if httpbody.IsTooLarge(err) {
			s.log.Warn("LSP request body too large")
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
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

	// Log incoming request with INFO level
	s.log.WithFields(logrus.Fields{
		"method":    request.Method,
		"id":        request.ID,
		"uri":       r.RemoteAddr,
		"body_size": len(body),
	}).Info("Incoming LSP request")

	// Handle different LSP methods
	response := s.handleLSPMethod(request)

	// JSON-RPC notifications omit "id"; the server must not send a response body (JSON-RPC 2.0).
	if request.ID == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Log response with INFO level
	s.log.WithFields(logrus.Fields{
		"method":     request.Method,
		"id":         request.ID,
		"has_error":  response.Error != nil,
		"has_result": response.Result != nil,
	}).Debug("LSP response prepared")

	// Send response
	w.Header().Set("Content-Type", "application/json")
	responseJSON, err := json.Marshal(response)
	if err != nil {
		s.log.Errorf("Failed to marshal response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(responseJSON)
}

// handleLSPMethod handles different LSP methods
func (s *LSPServer) handleLSPMethod(request LSPRequest) LSPServerResponse {
	s.log.WithFields(logrus.Fields{
		"method": request.Method,
		"id":     request.ID,
	}).Debug("Handling LSP method")

	switch request.Method {
	case "initialize":
		return s.handleInitialize(request)
	case "initialized":
		// Notification some clients send after initialize; HTTP bridge may POST it with an id.
		return LSPServerResponse{JSONRPC: "2.0", ID: request.ID, Result: nil}
	case "$/cancelRequest":
		// Cancellation is not implemented; acknowledge so clients do not treat the server as broken.
		return LSPServerResponse{JSONRPC: "2.0", ID: request.ID, Result: nil}
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
	case "textDocument/definition":
		return s.handleDefinition(request)
	case "textDocument/references":
		return s.handleReferences(request)
	case "textDocument/prepareRename":
		return s.handlePrepareRename(request)
	case "textDocument/rename":
		return s.handleRename(request)
	case "textDocument/documentSymbol":
		return s.handleDocumentSymbol(request)
	case "workspace/symbol":
		return s.handleWorkspaceSymbol(request)
	case "textDocument/formatting":
		return s.handleFormatting(request)
	case "textDocument/codeAction":
		return s.handleCodeAction(request)
	case "textDocument/codeLens":
		return s.handleCodeLens(request)
	case "textDocument/foldingRange":
		return s.handleFoldingRange(request)
	// Custom LLM debugging methods
	case "textDocument/debugInfo":
		return s.handleDebugInfo(request)
	case "textDocument/compilerState":
		return s.handleCompilerState(request)
	case "textDocument/phaseDetails":
		return s.handlePhaseDetails(request)
	case "shutdown":
		return s.handleShutdown(request)
	case "exit":
		return s.handleExit(request)
	default:
		entry := s.log.WithFields(logrus.Fields{
			"method": request.Method,
			"id":     request.ID,
		})
		if strings.HasPrefix(request.Method, "$/") {
			entry.Debug("Unsupported $/ LSP method")
		} else {
			entry.Warn("Unknown LSP method requested")
		}
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

// handlePublishDiagnostics handles the textDocument/publishDiagnostics method
func (s *LSPServer) handlePublishDiagnostics(request LSPRequest) LSPServerResponse {
	// This is typically a notification from the client, not a request
	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  nil,
	}
}

// compileForstFile compiles a Forst file and returns diagnostics
func (s *LSPServer) compileForstFile(filePath, content string, _ Debugger) []LSPDiagnostic {
	defer func() {
		if r := recover(); r != nil {
			s.log.Errorf("Panic in compileForstFile for %s: %v", filePath, r)
		}
	}()

	s.debugEvents = make([]DebugEvent, 0)
	if cd, ok := s.debugger.(*CompilerDebugger); ok {
		cd.ResetStructuredOutputs()
	}

	ctx, ok := s.analyzeForstContent(filePath, content)
	if !ok || ctx == nil {
		return nil
	}
	if ctx.ParseErr != nil {
		diagnostic := diagnosticForParseFailure(fileURIForLocalPath(filePath), content, ctx.ParseErr)
		parserDebugger := s.debugger.GetDebugger(PhaseParser, filePath)
		parserDebugger.LogError(EventParserError, "Parsing failed", &ErrorInfo{
			Code:     ErrorCodeInvalidSyntax,
			Message:  ctx.ParseErr.Error(),
			Severity: SeverityError,
		})
		return []LSPDiagnostic{diagnostic}
	}
	if ctx.CheckErr != nil {
		diagnostic := diagnosticForTypecheckError(fileURIForLocalPath(filePath), content, ctx.CheckErr, "forst-typechecker", ErrorCodeTypeMismatch)
		diagnostic.Message = fmt.Sprintf("Type checking error: %v", ctx.CheckErr)
		typecheckerDebugger := s.debugger.GetDebugger(PhaseTypechecker, filePath)
		typecheckerDebugger.LogError(EventTypeError, "Type checking failed", &ErrorInfo{
			Code:     ErrorCodeTypeMismatch,
			Message:  ctx.CheckErr.Error(),
			Severity: SeverityError,
		})
		if ctx.TC != nil {
			typecheckerDebugger.LogScope(EventScopeEntered, "Current scope at error", &ScopeInfo{
				Variables: convertVariableTypes(ctx.TC.VariableTypes),
				Types:   make(map[string]string),
				Stack:   []string{"global"},
			})
		}
		return []LSPDiagnostic{diagnostic}
	}
	return s.transformDiagnostics(ctx, filePath)
}

// compileForstFilePackageGroup runs typecheck/transform on the merged same-package AST when multiple
// open buffers belong to one package. Falls back to single-file compileForstFile when merge is unavailable.
func (s *LSPServer) compileForstFilePackageGroup(uri, filePath, content string) []LSPDiagnostic {
	defer func() {
		if r := recover(); r != nil {
			s.log.Errorf("Panic in compileForstFilePackageGroup for %s: %v", filePath, r)
		}
	}()

	snap, _, ok := s.analyzePackageGroupMerged(uri)
	if !ok {
		debugger := s.debugger.GetDebugger(PhaseParser, filePath)
		return s.compileForstFile(filePath, content, debugger)
	}
	if snap.checkErr != nil {
		d := diagnosticForTypecheckError(fileURIForLocalPath(filePath), content, snap.checkErr, "forst-typechecker", ErrorCodeTypeMismatch)
		d.Message = fmt.Sprintf("Type checking error: %v", snap.checkErr)
		return []LSPDiagnostic{d}
	}
	transformer := transformer_go.New(snap.tc, s.log, false)
	if _, err := transformer.TransformForstFileToGo(snap.mergedNodes); err != nil {
		d := diagnosticForTypecheckOrTransform(fileURIForLocalPath(filePath), content, err, "forst-transformer", ErrorCodeTransformationFailed)
		d.Message = fmt.Sprintf("Transformation error: %v", err)
		return []LSPDiagnostic{d}
	}
	return []LSPDiagnostic{}
}

// convertVariableTypes converts the typechecker's variable types to a string map for debugging
func convertVariableTypes(variableTypes map[ast.Identifier][]ast.TypeNode) map[string]string {
	result := make(map[string]string)
	for varName, types := range variableTypes {
		if len(types) > 0 {
			result[string(varName)] = types[0].String()
		}
	}
	return result
}

// sendDiagnosticsNotification sends a diagnostics notification
func (s *LSPServer) sendDiagnosticsNotification(uri string, diagnostics []LSPDiagnostic) {
	// In a real LSP implementation, this would be sent to the client
	// For now, we just log it
	s.log.Debugf("Sending diagnostics for %s: %d diagnostics", uri, len(diagnostics))
}

// handleDebugInfo handles the textDocument/debugInfo method primarily for LLM debugging
func (s *LSPServer) handleDebugInfo(request LSPRequest) LSPServerResponse {
	var params map[string]any
	if err := json.Unmarshal(request.Params, &params); err != nil {
		s.log.Errorf("Failed to parse debugInfo params: %v", err)
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &LSPError{
				Code:    -32602,
				Message: "Invalid params",
			},
		}
	}

	// Extract text document URI
	textDoc, ok := params["textDocument"].(map[string]any)
	if !ok {
		s.log.Error("textDocument not found in debugInfo params")
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &LSPError{
				Code:    -32602,
				Message: "textDocument not found",
			},
		}
	}

	uri, _ := textDoc["uri"].(string)
	if uri == "" {
		s.log.Error("textDocument.uri not found in debugInfo params")
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &LSPError{
				Code:    -32602,
				Message: "textDocument.uri not found",
			},
		}
	}

	// Extract compression parameter
	useCompression := true // Default to true
	if compressionParam, exists := params["compression"]; exists {
		if compressionBool, ok := compressionParam.(bool); ok {
			useCompression = compressionBool
		}
	} else {
		// No compression parameter provided, use default (true)
		useCompression = true
	}

	// Extract summary parameter
	summaryOnly := false
	if summaryParam, exists := params["summary"]; exists {
		if summaryBool, ok := summaryParam.(bool); ok {
			summaryOnly = summaryBool
		}
	}

	s.log.WithFields(logrus.Fields{
		"uri":          uri,
		"method":       "textDocument/debugInfo",
		"compression":  useCompression,
		"summary":      summaryOnly,
		"param_given":  params["compression"] != nil,
		"param_exists": params["compression"] != nil,
		"raw_params":   string(request.Params),
		"all_params":   params,
	}).Info("Debug info request received")

	// Get comprehensive debug information with compression option
	debugInfo := s.getComprehensiveDebugInfo(uri, useCompression, summaryOnly)

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  debugInfo,
	}
}

// handleCompilerState handles the textDocument/compilerState method
func (s *LSPServer) handleCompilerState(request LSPRequest) LSPServerResponse {
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

	// Get current compiler state
	compilerState := s.getCompilerState(params.TextDocument.URI)

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  compilerState,
	}
}

// handlePhaseDetails handles the textDocument/phaseDetails method
func (s *LSPServer) handlePhaseDetails(request LSPRequest) LSPServerResponse {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Phase string `json:"phase,omitempty"`
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

	// Get detailed phase information
	phaseDetails := s.getPhaseDetails(params.TextDocument.URI, params.Phase)

	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  phaseDetails,
	}
}

// getComprehensiveDebugInfo provides comprehensive debugging information for LLM debugging
func (s *LSPServer) getComprehensiveDebugInfo(uri string, useCompression bool, summaryOnly bool) map[string]any {
	// Extract file metadata once
	fileMetadata := s.extractFileMetadata(uri)

	// Register file in package store
	filePath := fileMetadata.Path
	packagePath := extractPackagePath(filePath)
	fileID := s.debugger.(*CompilerDebugger).packageStore.RegisterFile(filePath, packagePath)

	// Get all debug output from all phases
	allOutput, err := s.debugger.GetAllOutput()
	if err != nil {
		return map[string]any{
			"error": fmt.Sprintf("Failed to get debug output: %v", err),
		}
	}

	// Process debug events and convert to LSP structures
	_ = s.lspDebugger.ProcessDebugEvents()

	// Convert byte slices to structured data for better LLM consumption
	structuredOutputs := make(map[string]any)
	for phase, data := range allOutput {
		var events []DebugEvent
		if err := json.Unmarshal(data, &events); err == nil {
			// Optimize events by removing redundant file metadata
			optimizedEvents := s.optimizeDebugEvents(events, fileMetadata)
			structuredOutputs[string(phase)] = optimizedEvents
		} else {
			// Fallback to base64 if unmarshaling fails
			structuredOutputs[string(phase)] = map[string]any{
				"encoding": "base64",
				"data":     string(data),
				"error":    "Failed to parse as structured data",
			}
		}
	}

	// Build the debug info response with the expected structure
	debugInfo := map[string]any{
		"uri":       uri,
		"debugMode": s.debugMode,
		"timestamp": time.Now(),
		"file":      fileMetadata,
		"fileID":    fileID,
	}

	// Add package store information
	packageStore := s.debugger.(*CompilerDebugger).packageStore
	debugInfo["packageStore"] = map[string]any{
		"files":    packageStore.GetAllFiles(),
		"packages": packageStore.GetAllPackages(),
	}

	var rawOutput map[string]any
	if summaryOnly {
		// Summary mode: Just phase summaries (timing, event counts)
		rawOutput = map[string]any{
			"phaseSummaries": s.getPhaseSummaries(uri),
		}
	} else {
		// Full mode: Complete debugging information
		rawOutput = map[string]any{
			"phaseOutputs": structuredOutputs,
			"diagnostics":  s.lspDebugger.GetDiagnostics(),
		}
	}

	if useCompression {
		debugInfo["output"] = map[string]any{
			"encoding": map[string]any{
				"format":        "structured_json",
				"compression":   true,
				"llm_optimized": true,
			},
			"data": s.createCompressedDebugData(rawOutput),
		}
	} else {
		// No compression - include full phase outputs
		debugInfo["output"] = map[string]any{
			"encoding": map[string]any{
				"format":        "structured_json",
				"compression":   false,
				"llm_optimized": true,
			},
			"data": rawOutput,
		}
	}

	return debugInfo
}

// extractFileMetadata extracts file metadata from URI
func (s *LSPServer) extractFileMetadata(uri string) *FileMetadata {
	filePath := filePathFromDocumentURI(uri)

	// Extract filename
	parts := strings.Split(filePath, "/")
	filename := filePath
	if len(parts) > 0 {
		filename = parts[len(parts)-1]
	}

	return &FileMetadata{
		URI:      uri,
		Path:     filePath,
		Filename: filename,
	}
}

// optimizeDebugEvents removes redundant file metadata from debug events
func (s *LSPServer) optimizeDebugEvents(events []DebugEvent, _ *FileMetadata) []map[string]any {
	optimizedEvents := make([]map[string]any, 0, len(events))

	for _, event := range events {
		// Create optimized event without redundant file metadata
		optimizedEvent := map[string]any{
			"timestamp":  event.Timestamp,
			"phase":      event.Phase,
			"event_type": event.EventType,
			"message":    event.Message,
		}

		// Add optional fields only if they have values
		if event.Line > 0 {
			optimizedEvent["line"] = event.Line
		}
		if event.Function != "" {
			optimizedEvent["function"] = event.Function
		}
		if len(event.Data) > 0 {
			optimizedEvent["data"] = event.Data
		}
		if event.Scope != nil {
			optimizedEvent["scope"] = event.Scope
		}
		if event.AST != nil {
			optimizedEvent["ast"] = event.AST
		}
		if event.TypeInfo != nil {
			optimizedEvent["type_info"] = event.TypeInfo
		}
		if event.Error != nil {
			optimizedEvent["error"] = event.Error
		}

		optimizedEvents = append(optimizedEvents, optimizedEvent)
	}

	return optimizedEvents
}

// getPhaseSummaries returns performance metrics for each phase (timing, event counts)
func (s *LSPServer) getPhaseSummaries(uri string) map[string]any {
	filePath := filePathFromDocumentURI(uri)

	// Get debuggers for each phase
	lexerDebugger := s.debugger.GetDebugger(PhaseLexer, filePath)
	parserDebugger := s.debugger.GetDebugger(PhaseParser, filePath)
	typecheckerDebugger := s.debugger.GetDebugger(PhaseTypechecker, filePath)
	transformerDebugger := s.debugger.GetDebugger(PhaseTransformer, filePath)

	phaseSummaries := map[string]any{
		"lexer":       lexerDebugger.GetPhaseSummary(),
		"parser":      parserDebugger.GetPhaseSummary(),
		"typechecker": typecheckerDebugger.GetPhaseSummary(),
		"transformer": transformerDebugger.GetPhaseSummary(),
	}

	return phaseSummaries
}

// HandleDebugInfoDirect provides direct access to debugInfo functionality without HTTP overhead
func (s *LSPServer) HandleDebugInfoDirect(request LSPRequest, content string) LSPServerResponse {
	// Extract URI from debugInfo params
	var debugParams map[string]any
	if err := json.Unmarshal(request.Params, &debugParams); err != nil {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &LSPError{
				Code:    -32602,
				Message: "Failed to parse debugInfo params",
			},
		}
	}

	textDoc, ok := debugParams["textDocument"].(map[string]any)
	if !ok {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &LSPError{
				Code:    -32602,
				Message: "textDocument not found in params",
			},
		}
	}

	uri, ok := textDoc["uri"].(string)
	if !ok {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &LSPError{
				Code:    -32602,
				Message: "uri not found in textDocument",
			},
		}
	}

	// First, simulate a didOpen request to set up the file
	didOpenRequest := LSPRequest{
		JSONRPC: "2.0",
		ID:      "didOpen",
		Method:  "textDocument/didOpen",
		Params:  nil,
	}

	// Create didOpen params
	didOpenParams := map[string]any{
		"textDocument": map[string]any{
			"uri":     uri,
			"version": 1,
			"text":    content,
		},
	}

	// Marshal didOpen params
	didOpenParamsJSON, err := json.Marshal(didOpenParams)
	if err != nil {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &LSPError{
				Code:    -32602,
				Message: "Failed to marshal didOpen params",
			},
		}
	}
	didOpenRequest.Params = didOpenParamsJSON

	// Process the didOpen request
	s.handleDidOpen(didOpenRequest)

	// Now handle the debugInfo request
	return s.handleDebugInfo(request)
}

// createCompressedDebugData creates a compressed version of debug data
func (s *LSPServer) createCompressedDebugData(data any) map[string]any {
	// Marshal to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return map[string]any{
			"error": fmt.Sprintf("Failed to marshal data: %v", err),
		}
	}

	// Compress with gzip
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(jsonData); err != nil {
		return map[string]any{
			"error": fmt.Sprintf("Failed to compress data: %v", err),
		}
	}
	if err := gw.Close(); err != nil {
		return map[string]any{
			"error": fmt.Sprintf("Failed to finish gzip: %v", err),
		}
	}

	// Encode as base64 for JSON compatibility
	compressed := base64.StdEncoding.EncodeToString(buf.Bytes())

	originalSize := len(jsonData)
	compressedSize := len(compressed)
	compressionRatio := float64(compressedSize) / float64(originalSize)

	return map[string]any{
		"data":              compressed,
		"encoding":          "gzip+base64",
		"original_size":     originalSize,
		"compressed_size":   compressedSize,
		"compression_ratio": compressionRatio,
		"llm_optimized":     false, // Requires decoding
	}
}

// getCompilerState provides current compiler state information (types, variables, scopes)
func (s *LSPServer) getCompilerState(uri string) map[string]any {
	filePath := filePathFromDocumentURI(uri)

	// Get debugger for each phase
	lexerDebugger := s.debugger.GetDebugger(PhaseLexer, filePath)
	parserDebugger := s.debugger.GetDebugger(PhaseParser, filePath)
	typecheckerDebugger := s.debugger.GetDebugger(PhaseTypechecker, filePath)
	transformerDebugger := s.debugger.GetDebugger(PhaseTransformer, filePath)

	compilerState := map[string]any{
		"uri": uri,
		"state": map[string]any{
			"lexer": map[string]any{
				"tokens": getDebuggerOutput(lexerDebugger),
			},
			"parser": map[string]any{
				"ast": getDebuggerOutput(parserDebugger),
			},
			"typechecker": map[string]any{
				"types": getDebuggerOutput(typecheckerDebugger),
			},
			"transformer": map[string]any{
				"goCode": getDebuggerOutput(transformerDebugger),
			},
		},
		"debugMode": s.debugMode,
		"timestamp": time.Now(),
	}

	return compilerState
}

// getPhaseDetails provides detailed phase output information (tokens, AST, types, etc.)
func (s *LSPServer) getPhaseDetails(uri, phase string) map[string]any {
	filePath := filePathFromDocumentURI(uri)

	phaseDetails := map[string]any{
		"uri":       uri,
		"phase":     phase,
		"timestamp": time.Now(),
	}

	if phase == "" {
		// Return details for all phases
		phases := []CompilerPhase{PhaseLexer, PhaseParser, PhaseTypechecker, PhaseTransformer}
		allPhaseDetails := make(map[string]any)

		for _, p := range phases {
			debugger := s.debugger.GetDebugger(p, filePath)
			allPhaseDetails[string(p)] = map[string]any{
				"details": getDebuggerOutput(debugger),
			}
		}

		phaseDetails["phases"] = allPhaseDetails
	} else {
		// Return details for specific phase
		phaseEnum := CompilerPhase(phase)
		debugger := s.debugger.GetDebugger(phaseEnum, filePath)
		phaseDetails["details"] = getDebuggerOutput(debugger)
	}

	return phaseDetails
}

// getDebuggerOutput safely gets debugger output
func getDebuggerOutput(debugger Debugger) any {
	output, err := debugger.GetOutput()
	if err != nil {
		return map[string]any{
			"error": err.Error(),
		}
	}

	var events []DebugEvent
	if err := json.Unmarshal(output, &events); err != nil {
		return map[string]any{
			"error": fmt.Sprintf("Failed to unmarshal debug events: %v", err),
			"raw":   string(output),
		}
	}

	return events
}
