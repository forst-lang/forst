package lsp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"forst/internal/ast"
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
		debugMode:   true, // Enable debug mode by default for LLM debugging
		debugEvents: make([]DebugEvent, 0),
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
func (s *LSPServer) compileForstFile(filePath, content string, debugger Debugger) []LSPDiagnostic {
	// Add panic recovery to prevent LSP server crashes
	defer func() {
		if r := recover(); r != nil {
			s.log.Errorf("Panic in compileForstFile for %s: %v", filePath, r)
		}
	}()

	// Clear previous debug events for this file
	s.debugEvents = make([]DebugEvent, 0)

	// Lexical analysis
	lex := lexer.New([]byte(content), filePath, s.log)
	tokens := lex.Lex()

	// Log detailed lexer information
	lexerDebugger := s.debugger.GetDebugger(PhaseLexer, filePath)
	lexerDebugger.LogEvent(EventLexerComplete, "Lexical analysis completed", map[string]interface{}{
		"token_count": len(tokens),
		"file":        filePath,
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
		// Create detailed diagnostic for parsing error
		diagnostic := CreateTypeErrorDiagnostic(
			"file://"+filePath,
			1, // Default to line 1
			"",
			"",
			"parsing error",
		)
		diagnostic.Message = fmt.Sprintf("Parsing error: %v", err)
		diagnostic.Source = "forst-parser"
		diagnostic.Code = ErrorCodeInvalidSyntax

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
	if err := tc.CheckTypes(astNodes); err != nil {
		// Create detailed diagnostic for type checking error
		diagnostic := CreateTypeErrorDiagnostic(
			"file://"+filePath,
			1, // Default to line 1
			"",
			"",
			"type checking error",
		)
		diagnostic.Message = fmt.Sprintf("Type checking error: %v", err)
		diagnostic.Source = "forst-typechecker"
		diagnostic.Code = ErrorCodeTypeMismatch

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
		// Create detailed diagnostic for transformation error
		diagnostic := CreateTypeErrorDiagnostic(
			"file://"+filePath,
			1, // Default to line 1
			"",
			"",
			"transformation error",
		)
		diagnostic.Message = fmt.Sprintf("Transformation error: %v", err)
		diagnostic.Source = "forst-transformer"
		diagnostic.Code = ErrorCodeTransformationFailed

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
	s.lspDebugger.ProcessDebugEvents()
	return s.lspDebugger.GetDiagnostics()
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

// handleDebugInfo handles the textDocument/debugInfo method for LLM debugging
func (s *LSPServer) handleDebugInfo(request LSPRequest) LSPServerResponse {
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

	// Get comprehensive debug information
	debugInfo := s.getComprehensiveDebugInfo(params.TextDocument.URI)

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
func (s *LSPServer) getComprehensiveDebugInfo(uri string) map[string]interface{} {
	// Convert URI to file path
	filePath := strings.TrimPrefix(uri, "file://")
	if runtime.GOOS == "windows" {
		filePath = strings.TrimPrefix(filePath, "/")
	}

	// Get all debug output from all phases
	allOutput, err := s.debugger.GetAllOutput()
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Failed to get debug output: %v", err),
		}
	}

	// Process debug events and convert to LSP structures
	s.lspDebugger.ProcessDebugEvents()

	debugInfo := map[string]interface{}{
		"uri":           uri,
		"filePath":      filePath,
		"debugMode":     s.debugMode,
		"phaseOutputs":  allOutput,
		"diagnostics":   s.lspDebugger.GetDiagnostics(),
		"hovers":        s.lspDebugger.GetHovers(),
		"completions":   s.lspDebugger.GetCompletions(),
		"debugEvents":   s.debugEvents,
		"compilerState": s.getCompilerState(uri),
		"phaseDetails":  s.getPhaseDetails(uri, ""),
		"timestamp":     time.Now(),
	}

	return debugInfo
}

// getCompilerState provides current compiler state information
func (s *LSPServer) getCompilerState(uri string) map[string]interface{} {
	// Convert URI to file path
	filePath := strings.TrimPrefix(uri, "file://")
	if runtime.GOOS == "windows" {
		filePath = strings.TrimPrefix(filePath, "/")
	}

	// Get debugger for each phase
	lexerDebugger := s.debugger.GetDebugger(PhaseLexer, filePath)
	parserDebugger := s.debugger.GetDebugger(PhaseParser, filePath)
	typecheckerDebugger := s.debugger.GetDebugger(PhaseTypechecker, filePath)
	transformerDebugger := s.debugger.GetDebugger(PhaseTransformer, filePath)

	compilerState := map[string]interface{}{
		"uri": uri,
		"phases": map[string]interface{}{
			"lexer": map[string]interface{}{
				"summary": lexerDebugger.GetPhaseSummary(),
				"output":  getDebuggerOutput(lexerDebugger),
			},
			"parser": map[string]interface{}{
				"summary": parserDebugger.GetPhaseSummary(),
				"output":  getDebuggerOutput(parserDebugger),
			},
			"typechecker": map[string]interface{}{
				"summary": typecheckerDebugger.GetPhaseSummary(),
				"output":  getDebuggerOutput(typecheckerDebugger),
			},
			"transformer": map[string]interface{}{
				"summary": transformerDebugger.GetPhaseSummary(),
				"output":  getDebuggerOutput(transformerDebugger),
			},
		},
		"debugMode": s.debugMode,
		"timestamp": time.Now(),
	}

	return compilerState
}

// getPhaseDetails provides detailed information about a specific phase
func (s *LSPServer) getPhaseDetails(uri, phase string) map[string]interface{} {
	// Convert URI to file path
	filePath := strings.TrimPrefix(uri, "file://")
	if runtime.GOOS == "windows" {
		filePath = strings.TrimPrefix(filePath, "/")
	}

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
				"summary": debugger.GetPhaseSummary(),
				"output":  getDebuggerOutput(debugger),
			}
		}

		phaseDetails["phases"] = allPhaseDetails
	} else {
		// Return details for specific phase
		phaseEnum := CompilerPhase(phase)
		debugger := s.debugger.GetDebugger(phaseEnum, filePath)
		phaseDetails["summary"] = debugger.GetPhaseSummary()
		phaseDetails["output"] = getDebuggerOutput(debugger)
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
