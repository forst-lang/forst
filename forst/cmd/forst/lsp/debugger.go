package lsp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// CompilerPhase represents different phases of the compilation process.
// This type is used to identify which phase of compilation is being debugged.
type CompilerPhase string

const (
	// PhaseLexer represents the lexical analysis phase
	PhaseLexer CompilerPhase = "lexer"
	// PhaseParser represents the parsing phase
	PhaseParser CompilerPhase = "parser"
	// PhaseTypechecker represents the type checking phase
	PhaseTypechecker CompilerPhase = "typechecker"
	// PhaseTransformer represents the code transformation phase
	PhaseTransformer CompilerPhase = "transformer"
	// PhaseDiscovery represents the function discovery phase
	PhaseDiscovery CompilerPhase = "discovery"
	// PhaseExecutor represents the execution phase
	PhaseExecutor CompilerPhase = "executor"
)

// Debugger defines the interface for structured debugging operations.
// This interface allows for easy testing and mocking of debugger functionality.
type Debugger interface {
	// LogEvent logs a structured debug event with the given type, message, and data.
	LogEvent(eventType, message string, data map[string]interface{})
	// LogScope logs scope information with the given event type, message, and scope details.
	LogScope(eventType, message string, scope *ScopeInfo)
	// LogAST logs AST node information with the given event type, message, and AST details.
	LogAST(eventType, message string, ast *ASTInfo)
	// LogType logs type information with the given event type, message, and type details.
	LogType(eventType, message string, typeInfo *TypeInfo)
	// LogError logs error information with the given event type, message, and error details.
	LogError(eventType, message string, errorInfo *ErrorInfo)
	// GetOutput returns all debug events as JSON bytes.
	GetOutput() ([]byte, error)
	// GetPhaseSummary returns a summary of the phase execution as a map.
	GetPhaseSummary() map[string]interface{}
	// PrintSummary prints a human-readable summary of the phase.
	PrintSummary()
}

// CompilerDebuggerInterface defines the interface for managing multiple debuggers.
// This interface allows for easy testing and mocking of compiler debugger functionality.
type CompilerDebuggerInterface interface {
	// GetDebugger returns or creates a debugger for the specified phase and file path.
	GetDebugger(phase CompilerPhase, filePath string) Debugger
	// GetAllOutput returns all debug output from all phases as a map.
	GetAllOutput() (map[CompilerPhase][]byte, error)
	// PrintAllSummaries prints summaries for all phases.
	PrintAllSummaries()
	// LogWithStructuredDebug logs to both the existing logger and structured debugger.
	LogWithStructuredDebug(logger *logrus.Logger, level logrus.Level, phase CompilerPhase, filePath, eventType, message string, data map[string]interface{})
}

// StructuredDebugger provides machine-readable debug output for compiler phases.
// This type implements the Debugger interface and provides structured logging capabilities.
type StructuredDebugger struct {
	phase     CompilerPhase
	filePath  string
	startTime time.Time
	output    []DebugEvent
}

// DebugEvent represents a single debug event with structured data.
// This type is used to capture information about compiler events for debugging and analysis.
type DebugEvent struct {
	// Timestamp when the event occurred
	Timestamp time.Time `json:"timestamp"`
	// Phase of compilation when the event occurred
	Phase CompilerPhase `json:"phase"`
	// Type of event (e.g., "token_created", "function_entered", "type_inferred")
	EventType string `json:"event_type"`
	// Source file being processed (optional)
	File string `json:"file,omitempty"`
	// Line number in source file (optional)
	Line int `json:"line,omitempty"`
	// Function name where event occurred (optional)
	Function string `json:"function,omitempty"`
	// Human-readable message describing the event
	Message string `json:"message"`
	// Additional structured data for the event
	Data map[string]interface{} `json:"data,omitempty"`
	// Scope information if relevant to the event
	Scope *ScopeInfo `json:"scope,omitempty"`
	// AST information if relevant to the event
	AST *ASTInfo `json:"ast,omitempty"`
	// Type information if relevant to the event
	TypeInfo *TypeInfo `json:"type_info,omitempty"`
	// Error information if the event represents an error
	Error *ErrorInfo `json:"error,omitempty"`
}

// ScopeInfo provides information about the current scope.
// This type captures the state of variables, types, and function context during compilation.
type ScopeInfo struct {
	// Name of the current function being processed
	FunctionName string `json:"function_name,omitempty"`
	// Map of variable names to their types
	Variables map[string]string `json:"variables,omitempty"`
	// Map of type names to their definitions
	Types map[string]string `json:"types,omitempty"`
	// Stack of scope levels (e.g., ["global", "function", "block"])
	Stack []string `json:"stack,omitempty"`
}

// ASTInfo provides information about AST nodes.
// This type captures details about abstract syntax tree nodes for debugging.
type ASTInfo struct {
	// Type of the AST node (e.g., "FunctionNode", "VariableNode")
	NodeType string `json:"node_type"`
	// Unique identifier for the node (optional)
	NodeID string `json:"node_id,omitempty"`
	// Type of the parent node (optional)
	ParentType string `json:"parent_type,omitempty"`
	// List of child node types (optional)
	Children []string `json:"children,omitempty"`
	// Additional properties specific to the node type
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// TypeInfo provides information about type inference and checking.
// This type captures type-related information for debugging type system issues.
type TypeInfo struct {
	// Type that was expected in this context
	ExpectedType string `json:"expected_type,omitempty"`
	// Type that was inferred by the typechecker
	InferredType string `json:"inferred_type,omitempty"`
	// Actual type found or used
	ActualType string `json:"actual_type,omitempty"`
	// Map of type constraints applied
	TypeConstraints map[string]string `json:"type_constraints,omitempty"`
	// List of type-related errors encountered
	TypeErrors []string `json:"type_errors,omitempty"`
}

// ErrorInfo provides structured error information.
// This type captures detailed error information with suggestions for resolution.
type ErrorInfo struct {
	// Error code for categorization
	Code string `json:"code"`
	// Human-readable error message
	Message string `json:"message"`
	// Severity level (e.g., "error", "warning", "info")
	Severity string `json:"severity"`
	// List of suggestions for fixing the error
	Suggestions []string `json:"suggestions,omitempty"`
}

// NewStructuredDebugger creates a new structured debugger for a compiler phase.
// The debugger will collect events for the specified phase and file path.
func NewStructuredDebugger(phase CompilerPhase, filePath string) *StructuredDebugger {
	return &StructuredDebugger{
		phase:     phase,
		filePath:  filePath,
		startTime: time.Now(),
		output:    make([]DebugEvent, 0),
	}
}

// LogEvent logs a structured debug event.
// This method implements the Debugger interface.
func (d *StructuredDebugger) LogEvent(eventType, message string, data map[string]interface{}) {
	event := DebugEvent{
		Timestamp: time.Now(),
		Phase:     d.phase,
		EventType: eventType,
		File:      d.filePath,
		Message:   message,
		Data:      data,
	}
	d.output = append(d.output, event)
}

// LogScope logs scope information.
// This method implements the Debugger interface.
func (d *StructuredDebugger) LogScope(eventType, message string, scope *ScopeInfo) {
	event := DebugEvent{
		Timestamp: time.Now(),
		Phase:     d.phase,
		EventType: eventType,
		File:      d.filePath,
		Message:   message,
		Scope:     scope,
	}
	d.output = append(d.output, event)
}

// LogAST logs AST node information.
// This method implements the Debugger interface.
func (d *StructuredDebugger) LogAST(eventType, message string, ast *ASTInfo) {
	event := DebugEvent{
		Timestamp: time.Now(),
		Phase:     d.phase,
		EventType: eventType,
		File:      d.filePath,
		Message:   message,
		AST:       ast,
	}
	d.output = append(d.output, event)
}

// LogType logs type information.
// This method implements the Debugger interface.
func (d *StructuredDebugger) LogType(eventType, message string, typeInfo *TypeInfo) {
	event := DebugEvent{
		Timestamp: time.Now(),
		Phase:     d.phase,
		EventType: eventType,
		File:      d.filePath,
		Message:   message,
		TypeInfo:  typeInfo,
	}
	d.output = append(d.output, event)
}

// LogError logs error information.
// This method implements the Debugger interface.
func (d *StructuredDebugger) LogError(eventType, message string, errorInfo *ErrorInfo) {
	event := DebugEvent{
		Timestamp: time.Now(),
		Phase:     d.phase,
		EventType: eventType,
		File:      d.filePath,
		Message:   message,
		Error:     errorInfo,
	}
	d.output = append(d.output, event)
}

// GetOutput returns all debug events as JSON.
// This method implements the Debugger interface.
func (d *StructuredDebugger) GetOutput() ([]byte, error) {
	return json.MarshalIndent(d.output, "", "  ")
}

// GetPhaseSummary returns a summary of the phase execution.
// This method implements the Debugger interface.
func (d *StructuredDebugger) GetPhaseSummary() map[string]interface{} {
	duration := time.Since(d.startTime)

	eventCounts := make(map[string]int)
	for _, event := range d.output {
		eventCounts[event.EventType]++
	}

	return map[string]interface{}{
		"phase":        d.phase,
		"file":         d.filePath,
		"duration_ms":  duration.Milliseconds(),
		"total_events": len(d.output),
		"event_counts": eventCounts,
		"start_time":   d.startTime,
		"end_time":     time.Now(),
	}
}

// PrintSummary prints a human-readable summary of the phase.
// This method implements the Debugger interface.
func (d *StructuredDebugger) PrintSummary() {
	summary := d.GetPhaseSummary()
	fmt.Printf("=== %s Phase Summary ===\n", d.phase)
	fmt.Printf("File: %s\n", summary["file"])
	fmt.Printf("Duration: %dms\n", summary["duration_ms"])
	fmt.Printf("Total Events: %d\n", summary["total_events"])

	if eventCounts, ok := summary["event_counts"].(map[string]int); ok {
		fmt.Printf("Event Breakdown:\n")
		for eventType, count := range eventCounts {
			fmt.Printf("  %s: %d\n", eventType, count)
		}
	}
	fmt.Println()
}

// CompilerDebugger manages debuggers for all phases.
// This type implements the CompilerDebuggerInterface and provides centralized debugger management.
type CompilerDebugger struct {
	debuggers map[CompilerPhase]*StructuredDebugger
	enabled   bool
}

// NewCompilerDebugger creates a new compiler debugger.
// The enabled parameter controls whether debugging is active.
func NewCompilerDebugger(enabled bool) *CompilerDebugger {
	return &CompilerDebugger{
		debuggers: make(map[CompilerPhase]*StructuredDebugger),
		enabled:   enabled,
	}
}

// GetDebugger returns or creates a debugger for the specified phase.
// This method implements the CompilerDebuggerInterface.
func (cd *CompilerDebugger) GetDebugger(phase CompilerPhase, filePath string) Debugger {
	if !cd.enabled {
		return nil
	}

	if debugger, exists := cd.debuggers[phase]; exists {
		return debugger
	}

	debugger := NewStructuredDebugger(phase, filePath)
	cd.debuggers[phase] = debugger
	return debugger
}

// GetAllOutput returns all debug output from all phases.
// This method implements the CompilerDebuggerInterface.
func (cd *CompilerDebugger) GetAllOutput() (map[CompilerPhase][]byte, error) {
	output := make(map[CompilerPhase][]byte)

	for phase, debugger := range cd.debuggers {
		data, err := debugger.GetOutput()
		if err != nil {
			return nil, err
		}
		output[phase] = data
	}

	return output, nil
}

// PrintAllSummaries prints summaries for all phases.
// This method implements the CompilerDebuggerInterface.
func (cd *CompilerDebugger) PrintAllSummaries() {
	fmt.Println("=== Compiler Debug Summary ===")
	for _, debugger := range cd.debuggers {
		debugger.PrintSummary()
	}
}

// LogWithStructuredDebug integrates with existing logrus logger.
// This function provides a bridge between the existing logging system and the structured debugger.
// This method implements the CompilerDebuggerInterface.
func (cd *CompilerDebugger) LogWithStructuredDebug(logger *logrus.Logger, level logrus.Level, phase CompilerPhase, filePath, eventType, message string, data map[string]interface{}) {
	// Log to existing logger with structured fields
	logger.WithFields(logrus.Fields{
		"phase":      phase,
		"event_type": eventType,
		"file":       filePath,
		"data":       data,
	}).Log(level, message)

	// Also log to structured debugger for machine-readable output
	if debugger := cd.GetDebugger(phase, filePath); debugger != nil {
		debugger.LogEvent(eventType, message, data)
	}
}
