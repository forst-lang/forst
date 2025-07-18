package lsp

import (
	"fmt"
	"strings"
	"time"
)

// EventType constants provide predefined event types for consistent debugging.
// These constants help IntelliSense provide better autocomplete and prevent typos.
const (
	// Lexer event types
	EventTokenCreated   = "token_created"
	EventTokenProcessed = "token_processed"
	EventLexerError     = "lexer_error"
	EventLexerComplete  = "lexer_complete"

	// Parser event types
	EventNodeCreated    = "node_created"
	EventFunctionParsed = "function_parsed"
	EventTypeDefParsed  = "type_def_parsed"
	EventParserError    = "parser_error"
	EventParserComplete = "parser_complete"

	// Typechecker event types
	EventTypeInferred        = "type_inferred"
	EventTypeChecked         = "type_checked"
	EventTypeError           = "type_error"
	EventScopeEntered        = "scope_entered"
	EventScopeExited         = "scope_exited"
	EventTypecheckerComplete = "typechecker_complete"

	// Transformer event types
	EventFunctionTransformed  = "function_transformed"
	EventTypeEmitted          = "type_emitted"
	EventStatementTransformed = "statement_transformed"
	EventTransformerError     = "transformer_error"
	EventTransformerComplete  = "transformer_complete"

	// Discovery event types
	EventFunctionDiscovered = "function_discovered"
	EventTypeDiscovered     = "type_discovered"
	EventDiscoveryError     = "discovery_error"
	EventDiscoveryComplete  = "discovery_complete"

	// Executor event types
	EventFunctionExecuted  = "function_executed"
	EventExecutionError    = "execution_error"
	EventExecutionComplete = "execution_complete"
)

// ErrorCode constants provide predefined error codes for consistent error reporting.
// These constants help IntelliSense provide better autocomplete and prevent typos.
const (
	// Lexer error codes
	ErrorCodeInvalidToken       = "INVALID_TOKEN"
	ErrorCodeUnexpectedChar     = "UNEXPECTED_CHAR"
	ErrorCodeUnterminatedString = "UNTERMINATED_STRING"

	// Parser error codes
	ErrorCodeUnexpectedToken = "UNEXPECTED_TOKEN"
	ErrorCodeMissingToken    = "MISSING_TOKEN"
	ErrorCodeInvalidSyntax   = "INVALID_SYNTAX"

	// Typechecker error codes
	ErrorCodeTypeMismatch      = "TYPE_MISMATCH"
	ErrorCodeUndefinedType     = "UNDEFINED_TYPE"
	ErrorCodeUndefinedVariable = "UNDEFINED_VARIABLE"
	ErrorCodeInvalidOperation  = "INVALID_OPERATION"

	// Transformer error codes
	ErrorCodeTransformationFailed = "TRANSFORMATION_FAILED"
	ErrorCodeTypeEmissionFailed   = "TYPE_EMISSION_FAILED"
	ErrorCodeCodeGenerationFailed = "CODE_GENERATION_FAILED"
)

// Severity constants provide predefined severity levels for error reporting.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
	SeverityInfo    = "info"
	SeverityDebug   = "debug"
)

// DebugEventBuilder provides a fluent interface for building debug events.
// This type makes it easier to construct debug events with proper IntelliSense support.
type DebugEventBuilder struct {
	event DebugEvent
}

// NewDebugEventBuilder creates a new debug event builder for the specified phase.
func NewDebugEventBuilder(phase CompilerPhase, filePath string) *DebugEventBuilder {
	return &DebugEventBuilder{
		event: DebugEvent{
			Timestamp: time.Now(),
			Phase:     phase,
			File:      filePath,
		},
	}
}

// WithEventType sets the event type for the debug event.
func (b *DebugEventBuilder) WithEventType(eventType string) *DebugEventBuilder {
	b.event.EventType = eventType
	return b
}

// WithMessage sets the message for the debug event.
func (b *DebugEventBuilder) WithMessage(message string) *DebugEventBuilder {
	b.event.Message = message
	return b
}

// WithLine sets the line number for the debug event.
func (b *DebugEventBuilder) WithLine(line int) *DebugEventBuilder {
	b.event.Line = line
	return b
}

// WithFunction sets the function name for the debug event.
func (b *DebugEventBuilder) WithFunction(function string) *DebugEventBuilder {
	b.event.Function = function
	return b
}

// WithData sets the additional data for the debug event.
func (b *DebugEventBuilder) WithData(data map[string]interface{}) *DebugEventBuilder {
	b.event.Data = data
	return b
}

// WithScope sets the scope information for the debug event.
func (b *DebugEventBuilder) WithScope(scope *ScopeInfo) *DebugEventBuilder {
	b.event.Scope = scope
	return b
}

// WithAST sets the AST information for the debug event.
func (b *DebugEventBuilder) WithAST(ast *ASTInfo) *DebugEventBuilder {
	b.event.AST = ast
	return b
}

// WithTypeInfo sets the type information for the debug event.
func (b *DebugEventBuilder) WithTypeInfo(typeInfo *TypeInfo) *DebugEventBuilder {
	b.event.TypeInfo = typeInfo
	return b
}

// WithError sets the error information for the debug event.
func (b *DebugEventBuilder) WithError(errorInfo *ErrorInfo) *DebugEventBuilder {
	b.event.Error = errorInfo
	return b
}

// Build returns the constructed debug event.
func (b *DebugEventBuilder) Build() DebugEvent {
	return b.event
}

// ScopeInfoBuilder provides a fluent interface for building scope information.
type ScopeInfoBuilder struct {
	scope ScopeInfo
}

// NewScopeInfoBuilder creates a new scope info builder.
func NewScopeInfoBuilder() *ScopeInfoBuilder {
	return &ScopeInfoBuilder{
		scope: ScopeInfo{
			Variables: make(map[string]string),
			Types:     make(map[string]string),
			Stack:     make([]string, 0),
		},
	}
}

// WithFunctionName sets the function name for the scope.
func (b *ScopeInfoBuilder) WithFunctionName(functionName string) *ScopeInfoBuilder {
	b.scope.FunctionName = functionName
	return b
}

// WithVariable adds a variable to the scope.
func (b *ScopeInfoBuilder) WithVariable(name, typeName string) *ScopeInfoBuilder {
	b.scope.Variables[name] = typeName
	return b
}

// WithType adds a type to the scope.
func (b *ScopeInfoBuilder) WithType(name, definition string) *ScopeInfoBuilder {
	b.scope.Types[name] = definition
	return b
}

// WithStackLevel adds a level to the scope stack.
func (b *ScopeInfoBuilder) WithStackLevel(level string) *ScopeInfoBuilder {
	b.scope.Stack = append(b.scope.Stack, level)
	return b
}

// Build returns the constructed scope info.
func (b *ScopeInfoBuilder) Build() *ScopeInfo {
	return &b.scope
}

// ErrorInfoBuilder provides a fluent interface for building error information.
type ErrorInfoBuilder struct {
	error ErrorInfo
}

// NewErrorInfoBuilder creates a new error info builder.
func NewErrorInfoBuilder(code, message string) *ErrorInfoBuilder {
	return &ErrorInfoBuilder{
		error: ErrorInfo{
			Code:        code,
			Message:     message,
			Severity:    SeverityError,
			Suggestions: make([]string, 0),
		},
	}
}

// WithSeverity sets the severity level for the error.
func (b *ErrorInfoBuilder) WithSeverity(severity string) *ErrorInfoBuilder {
	b.error.Severity = severity
	return b
}

// WithSuggestion adds a suggestion for fixing the error.
func (b *ErrorInfoBuilder) WithSuggestion(suggestion string) *ErrorInfoBuilder {
	b.error.Suggestions = append(b.error.Suggestions, suggestion)
	return b
}

// Build returns the constructed error info.
func (b *ErrorInfoBuilder) Build() *ErrorInfo {
	return &b.error
}

// Utility functions for common debugging operations

// CreateTokenEvent creates a debug event for token creation.
func CreateTokenEvent(phase CompilerPhase, filePath, tokenType, value string, line, column int) DebugEvent {
	return NewDebugEventBuilder(phase, filePath).
		WithEventType(EventTokenCreated).
		WithMessage(fmt.Sprintf("Created %s token: %s", tokenType, value)).
		WithLine(line).
		WithData(map[string]interface{}{
			"token_type": tokenType,
			"value":      value,
			"line":       line,
			"column":     column,
		}).
		Build()
}

// CreateFunctionEvent creates a debug event for function processing.
func CreateFunctionEvent(phase CompilerPhase, filePath, functionName string, parameters, returnTypes []string) DebugEvent {
	return NewDebugEventBuilder(phase, filePath).
		WithEventType(EventFunctionParsed).
		WithMessage(fmt.Sprintf("Processed function: %s", functionName)).
		WithFunction(functionName).
		WithData(map[string]interface{}{
			"function_name": functionName,
			"parameters":    parameters,
			"return_types":  returnTypes,
		}).
		Build()
}

// CreateTypeErrorEvent creates a debug event for type errors.
func CreateTypeErrorEvent(phase CompilerPhase, filePath, expectedType, actualType, context string) DebugEvent {
	errorInfo := NewErrorInfoBuilder(ErrorCodeTypeMismatch,
		fmt.Sprintf("Type mismatch: expected %s, got %s", expectedType, actualType)).
		WithSeverity(SeverityError).
		WithSuggestion(fmt.Sprintf("Ensure the variable is of type %s", expectedType)).
		WithSuggestion("Check type annotations and constraints").
		Build()

	return NewDebugEventBuilder(phase, filePath).
		WithEventType(EventTypeError).
		WithMessage(fmt.Sprintf("Type error in %s: expected %s, got %s", context, expectedType, actualType)).
		WithError(errorInfo).
		WithTypeInfo(&TypeInfo{
			ExpectedType: expectedType,
			ActualType:   actualType,
			TypeErrors:   []string{fmt.Sprintf("Expected %s, got %s", expectedType, actualType)},
		}).
		Build()
}

// CreateScopeEvent creates a debug event for scope changes.
func CreateScopeEvent(phase CompilerPhase, filePath, eventType, functionName string, variables map[string]string) DebugEvent {
	scopeInfo := NewScopeInfoBuilder().
		WithFunctionName(functionName).
		Build()

	for name, typeName := range variables {
		scopeInfo.Variables[name] = typeName
	}

	return NewDebugEventBuilder(phase, filePath).
		WithEventType(eventType).
		WithMessage(fmt.Sprintf("Scope %s for function: %s", eventType, functionName)).
		WithFunction(functionName).
		WithScope(scopeInfo).
		Build()
}

// FormatDebugEvent formats a debug event as a human-readable string.
func FormatDebugEvent(event DebugEvent) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("[%s]", event.Phase))
	parts = append(parts, fmt.Sprintf("[%s]", event.EventType))

	if event.File != "" {
		parts = append(parts, fmt.Sprintf("file=%s", event.File))
	}

	if event.Line > 0 {
		parts = append(parts, fmt.Sprintf("line=%d", event.Line))
	}

	if event.Function != "" {
		parts = append(parts, fmt.Sprintf("function=%s", event.Function))
	}

	parts = append(parts, event.Message)

	return strings.Join(parts, " ")
}

// ValidateEventType validates that an event type is one of the predefined constants.
func ValidateEventType(eventType string) error {
	validTypes := []string{
		EventTokenCreated, EventTokenProcessed, EventLexerError, EventLexerComplete,
		EventNodeCreated, EventFunctionParsed, EventTypeDefParsed, EventParserError, EventParserComplete,
		EventTypeInferred, EventTypeChecked, EventTypeError, EventScopeEntered, EventScopeExited, EventTypecheckerComplete,
		EventFunctionTransformed, EventTypeEmitted, EventStatementTransformed, EventTransformerError, EventTransformerComplete,
		EventFunctionDiscovered, EventTypeDiscovered, EventDiscoveryError, EventDiscoveryComplete,
		EventFunctionExecuted, EventExecutionError, EventExecutionComplete,
	}

	for _, validType := range validTypes {
		if eventType == validType {
			return nil
		}
	}

	return fmt.Errorf("invalid event type: %s", eventType)
}

// ValidateErrorCode validates that an error code is one of the predefined constants.
func ValidateErrorCode(errorCode string) error {
	validCodes := []string{
		ErrorCodeInvalidToken, ErrorCodeUnexpectedChar, ErrorCodeUnterminatedString,
		ErrorCodeUnexpectedToken, ErrorCodeMissingToken, ErrorCodeInvalidSyntax,
		ErrorCodeTypeMismatch, ErrorCodeUndefinedType, ErrorCodeUndefinedVariable, ErrorCodeInvalidOperation,
		ErrorCodeTransformationFailed, ErrorCodeTypeEmissionFailed, ErrorCodeCodeGenerationFailed,
	}

	for _, validCode := range validCodes {
		if errorCode == validCode {
			return nil
		}
	}

	return fmt.Errorf("invalid error code: %s", errorCode)
}
