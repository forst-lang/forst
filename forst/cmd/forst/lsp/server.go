package lsp

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"forst/internal/ast"
	"forst/internal/goload"
	"forst/internal/lexer"
	"forst/internal/parser"
	transformer_go "forst/internal/transformer/go"
	"forst/internal/typechecker"

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
		debugger:      debugger,
		lspDebugger:   lspDebugger,
		log:           log,
		port:          port,
		debugMode:     true, // Enable debug mode by default for LLM debugging
		debugEvents:   make([]DebugEvent, 0),
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
	// Add panic recovery to prevent LSP server crashes
	defer func() {
		if r := recover(); r != nil {
			s.log.Errorf("Panic in compileForstFile for %s: %v", filePath, r)
		}
	}()

	// Clear previous debug events for this file
	s.debugEvents = make([]DebugEvent, 0)
	if cd, ok := s.debugger.(*CompilerDebugger); ok {
		cd.ResetStructuredOutputs()
	}

	// Get file ID from package store
	packageStore := s.debugger.(*CompilerDebugger).packageStore
	fileID := packageStore.RegisterFile(filePath, extractPackagePath(filePath))

	// Lexical analysis
	lex := lexer.New([]byte(content), string(fileID), s.log)
	tokens := lex.Lex()

	// Log detailed lexer information
	lexerDebugger := s.debugger.GetDebugger(PhaseLexer, filePath)
	lexerDebugger.LogEvent(EventLexerComplete, "Lexical analysis completed", map[string]interface{}{
		"token_count": len(tokens),
		"file_id":     fileID,
		"tokens":      tokens,
	})

	// Capture lexer debug events
	if lexerOutput, err := lexerDebugger.GetOutput(); err == nil {
		var lexerEvents []DebugEvent
		if json.Unmarshal(lexerOutput, &lexerEvents) == nil {
			s.debugEvents = append(s.debugEvents, lexerEvents...)
		}
	}

	// Parsing with panic recovery
	psr := parser.New(tokens, string(fileID), s.log)
	var astNodes []ast.Node
	var err error

	func() {
		defer func() {
			if r := recover(); r != nil {
				s.log.Errorf("Parser panic for %s: %v", filePath, r)
				if pe, ok := r.(*parser.ParseError); ok {
					// Preserve *ParseError so LSP diagnostics get token line/column (fmt.Errorf would stringify it).
					err = pe
				} else {
					err = fmt.Errorf("parser panic: %v", r)
				}
			}
		}()
		astNodes, err = psr.ParseFile()
	}()

	if err != nil {
		diagnostic := diagnosticForParseFailure(fileURIForLocalPath(filePath), content, err)

		// Log parser error event
		parserDebugger := s.debugger.GetDebugger(PhaseParser, filePath)
		parserDebugger.LogError(EventParserError, "Parsing failed", &ErrorInfo{
			Code:     ErrorCodeInvalidSyntax,
			Message:  err.Error(),
			Severity: SeverityError,
			Suggestions: []string{
				"Check syntax for missing brackets, parentheses, or semicolons",
				"Verify all keywords are properly spelled",
				"Ensure proper indentation and structure",
			},
		})

		return []LSPDiagnostic{diagnostic}
	}

	// Log detailed parser information
	parserDebugger := s.debugger.GetDebugger(PhaseParser, filePath)
	parserDebugger.LogEvent(EventParserComplete, "Parsing completed", map[string]interface{}{
		"node_count": len(astNodes),
		"file":       filePath,
		"ast_nodes":  astNodes,
	})

	// Capture parser debug events
	if parserOutput, err := parserDebugger.GetOutput(); err == nil {
		var parserEvents []DebugEvent
		if json.Unmarshal(parserOutput, &parserEvents) == nil {
			s.debugEvents = append(s.debugEvents, parserEvents...)
		}
	}

	// Type checking with detailed error capture
	tc := typechecker.New(s.log, false)
	tc.GoWorkspaceDir = goload.FindModuleRoot(filepath.Dir(filePath))
	if err := tc.CheckTypes(astNodes); err != nil {
		diagnostic := diagnosticForTypecheckError(fileURIForLocalPath(filePath), content, err, "forst-typechecker", ErrorCodeTypeMismatch)
		diagnostic.Message = fmt.Sprintf("Type checking error: %v", err)

		// Log typechecker error event with detailed information
		typecheckerDebugger := s.debugger.GetDebugger(PhaseTypechecker, filePath)
		typecheckerDebugger.LogError(EventTypeError, "Type checking failed", &ErrorInfo{
			Code:     ErrorCodeTypeMismatch,
			Message:  err.Error(),
			Severity: SeverityError,
			Suggestions: []string{
				"Check variable types and declarations",
				"Verify function signatures match call sites",
				"Ensure all types are properly defined",
				"Check for type assertion issues",
			},
		})

		// Add typechecker state information
		typecheckerDebugger.LogScope(EventScopeEntered, "Current scope at error", &ScopeInfo{
			FunctionName: "unknown",
			Variables:    convertVariableTypes(tc.VariableTypes),
			Types:        make(map[string]string),
			Stack:        []string{"global"},
		})

		return []LSPDiagnostic{diagnostic}
	}

	// Log detailed typechecker information
	typecheckerDebugger := s.debugger.GetDebugger(PhaseTypechecker, filePath)
	typecheckerDebugger.LogEvent(EventTypecheckerComplete, "Type checking completed", map[string]interface{}{
		"file":           filePath,
		"inferred_types": tc.InferredTypes,
		"variable_types": convertVariableTypes(tc.VariableTypes),
		"function_types": tc.FunctionReturnTypes,
		"type_defs":      tc.Defs,
	})

	// Capture typechecker debug events
	if typecheckerOutput, err := typecheckerDebugger.GetOutput(); err == nil {
		var typecheckerEvents []DebugEvent
		if json.Unmarshal(typecheckerOutput, &typecheckerEvents) == nil {
			s.debugEvents = append(s.debugEvents, typecheckerEvents...)
		}
	}

	// Code transformation with detailed error capture
	transformer := transformer_go.New(tc, s.log, false)
	_, err = transformer.TransformForstFileToGo(astNodes)
	if err != nil {
		diagnostic := diagnosticForTypecheckOrTransform(fileURIForLocalPath(filePath), content, err, "forst-transformer", ErrorCodeTransformationFailed)
		diagnostic.Message = fmt.Sprintf("Transformation error: %v", err)

		// Log transformer error event
		transformerDebugger := s.debugger.GetDebugger(PhaseTransformer, filePath)
		transformerDebugger.LogError(EventTransformerError, "Code transformation failed", &ErrorInfo{
			Code:     ErrorCodeTransformationFailed,
			Message:  err.Error(),
			Severity: SeverityError,
			Suggestions: []string{
				"Check for unsupported language constructs",
				"Verify type definitions are complete",
				"Ensure all referenced types are defined",
				"Check for recursive type definitions",
			},
		})

		return []LSPDiagnostic{diagnostic}
	}

	// Log detailed transformer information
	transformerDebugger := s.debugger.GetDebugger(PhaseTransformer, filePath)
	transformerDebugger.LogEvent(EventTransformerComplete, "Code transformation completed", map[string]interface{}{
		"file":               filePath,
		"transformer_status": "completed",
	})

	// Capture transformer debug events
	if transformerOutput, err := transformerDebugger.GetOutput(); err == nil {
		var transformerEvents []DebugEvent
		if json.Unmarshal(transformerOutput, &transformerEvents) == nil {
			s.debugEvents = append(s.debugEvents, transformerEvents...)
		}
	}

	// Process debug events and convert to LSP diagnostics
	_ = s.lspDebugger.ProcessDebugEvents()
	return s.lspDebugger.GetDiagnostics()
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
	var params map[string]interface{}
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
	textDoc, ok := params["textDocument"].(map[string]interface{})
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
func (s *LSPServer) getComprehensiveDebugInfo(uri string, useCompression bool, summaryOnly bool) map[string]interface{} {
	// Extract file metadata once
	fileMetadata := s.extractFileMetadata(uri)

	// Register file in package store
	filePath := fileMetadata.Path
	packagePath := extractPackagePath(filePath)
	fileID := s.debugger.(*CompilerDebugger).packageStore.RegisterFile(filePath, packagePath)

	// Get all debug output from all phases
	allOutput, err := s.debugger.GetAllOutput()
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to get debug output: %v", err),
		}
	}

	// Process debug events and convert to LSP structures
	_ = s.lspDebugger.ProcessDebugEvents()

	// Convert byte slices to structured data for better LLM consumption
	structuredOutputs := make(map[string]interface{})
	for phase, data := range allOutput {
		var events []DebugEvent
		if err := json.Unmarshal(data, &events); err == nil {
			// Optimize events by removing redundant file metadata
			optimizedEvents := s.optimizeDebugEvents(events, fileMetadata)
			structuredOutputs[string(phase)] = optimizedEvents
		} else {
			// Fallback to base64 if unmarshaling fails
			structuredOutputs[string(phase)] = map[string]interface{}{
				"encoding": "base64",
				"data":     string(data),
				"error":    "Failed to parse as structured data",
			}
		}
	}

	// Build the debug info response with the expected structure
	debugInfo := map[string]interface{}{
		"uri":       uri,
		"debugMode": s.debugMode,
		"timestamp": time.Now(),
		"file":      fileMetadata,
		"fileID":    fileID,
	}

	// Add package store information
	packageStore := s.debugger.(*CompilerDebugger).packageStore
	debugInfo["packageStore"] = map[string]interface{}{
		"files":    packageStore.GetAllFiles(),
		"packages": packageStore.GetAllPackages(),
	}

	var rawOutput map[string]interface{}
	if summaryOnly {
		// Summary mode: Just phase summaries (timing, event counts)
		rawOutput = map[string]interface{}{
			"phaseSummaries": s.getPhaseSummaries(uri),
		}
	} else {
		// Full mode: Complete debugging information
		rawOutput = map[string]interface{}{
			"phaseOutputs": structuredOutputs,
			"diagnostics":  s.lspDebugger.GetDiagnostics(),
		}
	}

	if useCompression {
		debugInfo["output"] = map[string]interface{}{
			"encoding": map[string]interface{}{
				"format":        "structured_json",
				"compression":   true,
				"llm_optimized": true,
			},
			"data": s.createCompressedDebugData(rawOutput),
		}
	} else {
		// No compression - include full phase outputs
		debugInfo["output"] = map[string]interface{}{
			"encoding": map[string]interface{}{
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
func (s *LSPServer) optimizeDebugEvents(events []DebugEvent, _ *FileMetadata) []map[string]interface{} {
	optimizedEvents := make([]map[string]interface{}, 0, len(events))

	for _, event := range events {
		// Create optimized event without redundant file metadata
		optimizedEvent := map[string]interface{}{
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
func (s *LSPServer) getPhaseSummaries(uri string) map[string]interface{} {
	filePath := filePathFromDocumentURI(uri)

	// Get debuggers for each phase
	lexerDebugger := s.debugger.GetDebugger(PhaseLexer, filePath)
	parserDebugger := s.debugger.GetDebugger(PhaseParser, filePath)
	typecheckerDebugger := s.debugger.GetDebugger(PhaseTypechecker, filePath)
	transformerDebugger := s.debugger.GetDebugger(PhaseTransformer, filePath)

	phaseSummaries := map[string]interface{}{
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
	var debugParams map[string]interface{}
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

	textDoc, ok := debugParams["textDocument"].(map[string]interface{})
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
	didOpenParams := map[string]interface{}{
		"textDocument": map[string]interface{}{
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
func (s *LSPServer) createCompressedDebugData(data interface{}) map[string]interface{} {
	// Marshal to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to marshal data: %v", err),
		}
	}

	// Compress with gzip
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(jsonData); err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to compress data: %v", err),
		}
	}
	if err := gw.Close(); err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to finish gzip: %v", err),
		}
	}

	// Encode as base64 for JSON compatibility
	compressed := base64.StdEncoding.EncodeToString(buf.Bytes())

	originalSize := len(jsonData)
	compressedSize := len(compressed)
	compressionRatio := float64(compressedSize) / float64(originalSize)

	return map[string]interface{}{
		"data":              compressed,
		"encoding":          "gzip+base64",
		"original_size":     originalSize,
		"compressed_size":   compressedSize,
		"compression_ratio": compressionRatio,
		"llm_optimized":     false, // Requires decoding
	}
}

// getCompilerState provides current compiler state information (types, variables, scopes)
func (s *LSPServer) getCompilerState(uri string) map[string]interface{} {
	filePath := filePathFromDocumentURI(uri)

	// Get debugger for each phase
	lexerDebugger := s.debugger.GetDebugger(PhaseLexer, filePath)
	parserDebugger := s.debugger.GetDebugger(PhaseParser, filePath)
	typecheckerDebugger := s.debugger.GetDebugger(PhaseTypechecker, filePath)
	transformerDebugger := s.debugger.GetDebugger(PhaseTransformer, filePath)

	compilerState := map[string]interface{}{
		"uri": uri,
		"state": map[string]interface{}{
			"lexer": map[string]interface{}{
				"tokens": getDebuggerOutput(lexerDebugger),
			},
			"parser": map[string]interface{}{
				"ast": getDebuggerOutput(parserDebugger),
			},
			"typechecker": map[string]interface{}{
				"types": getDebuggerOutput(typecheckerDebugger),
			},
			"transformer": map[string]interface{}{
				"goCode": getDebuggerOutput(transformerDebugger),
			},
		},
		"debugMode": s.debugMode,
		"timestamp": time.Now(),
	}

	return compilerState
}

// getPhaseDetails provides detailed phase output information (tokens, AST, types, etc.)
func (s *LSPServer) getPhaseDetails(uri, phase string) map[string]interface{} {
	filePath := filePathFromDocumentURI(uri)

	phaseDetails := map[string]interface{}{
		"uri":       uri,
		"phase":     phase,
		"timestamp": time.Now(),
	}

	if phase == "" {
		// Return details for all phases
		phases := []CompilerPhase{PhaseLexer, PhaseParser, PhaseTypechecker, PhaseTransformer}
		allPhaseDetails := make(map[string]interface{})

		for _, p := range phases {
			debugger := s.debugger.GetDebugger(p, filePath)
			allPhaseDetails[string(p)] = map[string]interface{}{
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
func getDebuggerOutput(debugger Debugger) interface{} {
	output, err := debugger.GetOutput()
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	var events []DebugEvent
	if err := json.Unmarshal(output, &events); err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to unmarshal debug events: %v", err),
			"raw":   string(output),
		}
	}

	return events
}
