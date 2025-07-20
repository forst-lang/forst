package lsp

import (
	"encoding/json"
	"fmt"
	"time"
)

// LSPDiagnostic represents a diagnostic message compatible with LSP.
// This follows the LSP specification for diagnostic messages.
type LSPDiagnostic struct {
	// Range specifies the range within a document where the diagnostic applies.
	Range LSPRange `json:"range"`
	// Severity indicates the severity of the diagnostic.
	Severity LSPDiagnosticSeverity `json:"severity"`
	// Code is the diagnostic's code, which might appear in the user interface.
	Code string `json:"code,omitempty"`
	// Source is a human-readable string describing the source of this diagnostic.
	Source string `json:"source,omitempty"`
	// Message is the diagnostic's message.
	Message string `json:"message"`
	// Tags is an array of related diagnostic information.
	Tags []LSPDiagnosticTag `json:"tags,omitempty"`
	// RelatedInformation is an array of related diagnostic information.
	RelatedInformation []LSPDiagnosticRelatedInformation `json:"relatedInformation,omitempty"`
}

// LSPRange represents a range in a text document.
type LSPRange struct {
	// Start is the range's start position.
	Start LSPPosition `json:"start"`
	// End is the range's end position.
	End LSPPosition `json:"end"`
}

// LSPPosition represents a position in a text document.
type LSPPosition struct {
	// Line is the zero-based line value.
	Line int `json:"line"`
	// Character is the zero-based character value.
	Character int `json:"character"`
}

// LSPDiagnosticSeverity represents the severity of a diagnostic.
type LSPDiagnosticSeverity int

const (
	// LSPDiagnosticSeverityError indicates an error.
	LSPDiagnosticSeverityError LSPDiagnosticSeverity = 1
	// LSPDiagnosticSeverityWarning indicates a warning.
	LSPDiagnosticSeverityWarning LSPDiagnosticSeverity = 2
	// LSPDiagnosticSeverityInformation indicates an informational message.
	LSPDiagnosticSeverityInformation LSPDiagnosticSeverity = 3
	// LSPDiagnosticSeverityHint indicates a hint.
	LSPDiagnosticSeverityHint LSPDiagnosticSeverity = 4
)

// LSPDiagnosticTag represents a diagnostic tag.
type LSPDiagnosticTag int

const (
	// LSPDiagnosticTagUnnecessary indicates that the diagnostic is unnecessary.
	LSPDiagnosticTagUnnecessary LSPDiagnosticTag = 1
	// LSPDiagnosticTagDeprecated indicates that the diagnostic is deprecated.
	LSPDiagnosticTagDeprecated LSPDiagnosticTag = 2
)

// LSPDiagnosticRelatedInformation represents related diagnostic information.
type LSPDiagnosticRelatedInformation struct {
	// Location is the location of this related diagnostic information.
	Location LSPLocation `json:"location"`
	// Message is the message of this related diagnostic information.
	Message string `json:"message"`
}

// LSPLocation represents a location inside a resource.
type LSPLocation struct {
	// URI is the document URI.
	URI string `json:"uri"`
	// Range is the document range.
	Range LSPRange `json:"range"`
}

// LSPHover represents the result of a hover request.
type LSPHover struct {
	// Contents is the hover's content.
	Contents LSPMarkedString `json:"contents"`
	// Range is an optional range is a range inside a text document.
	Range *LSPRange `json:"range,omitempty"`
}

// LSPMarkedString represents a string value which content can be represented as plain text.
type LSPMarkedString struct {
	// Language is the language identifier.
	Language string `json:"language,omitempty"`
	// Value is the actual value.
	Value string `json:"value"`
}

// LSPCompletionItem represents a completion item.
type LSPCompletionItem struct {
	// Label is the display text for the completion item.
	Label string `json:"label"`
	// Kind is the kind of this completion item.
	Kind LSPCompletionItemKind `json:"kind,omitempty"`
	// Detail is additional details for the completion item.
	Detail string `json:"detail,omitempty"`
	// Documentation is documentation for the completion item.
	Documentation string `json:"documentation,omitempty"`
	// SortText is a string that should be used when comparing this item with other items.
	SortText string `json:"sortText,omitempty"`
	// FilterText is a string that should be used when filtering a set of completion items.
	FilterText string `json:"filterText,omitempty"`
	// InsertText is the text to insert.
	InsertText string `json:"insertText,omitempty"`
	// InsertTextFormat is the format of the insert text.
	InsertTextFormat LSPInsertTextFormat `json:"insertTextFormat,omitempty"`
}

// LSPCompletionItemKind represents the kind of a completion item.
type LSPCompletionItemKind int

const (
	// LSPCompletionItemKindText indicates a text completion item.
	LSPCompletionItemKindText LSPCompletionItemKind = 1
	// LSPCompletionItemKindMethod indicates a method completion item.
	LSPCompletionItemKindMethod LSPCompletionItemKind = 2
	// LSPCompletionItemKindFunction indicates a function completion item.
	LSPCompletionItemKindFunction LSPCompletionItemKind = 3
	// LSPCompletionItemKindConstructor indicates a constructor completion item.
	LSPCompletionItemKindConstructor LSPCompletionItemKind = 4
	// LSPCompletionItemKindField indicates a field completion item.
	LSPCompletionItemKindField LSPCompletionItemKind = 5
	// LSPCompletionItemKindVariable indicates a variable completion item.
	LSPCompletionItemKindVariable LSPCompletionItemKind = 6
	// LSPCompletionItemKindClass indicates a class completion item.
	LSPCompletionItemKindClass LSPCompletionItemKind = 7
	// LSPCompletionItemKindInterface indicates an interface completion item.
	LSPCompletionItemKindInterface LSPCompletionItemKind = 8
	// LSPCompletionItemKindModule indicates a module completion item.
	LSPCompletionItemKindModule LSPCompletionItemKind = 9
	// LSPCompletionItemKindProperty indicates a property completion item.
	LSPCompletionItemKindProperty LSPCompletionItemKind = 10
	// LSPCompletionItemKindUnit indicates a unit completion item.
	LSPCompletionItemKindUnit LSPCompletionItemKind = 11
	// LSPCompletionItemKindValue indicates a value completion item.
	LSPCompletionItemKindValue LSPCompletionItemKind = 12
	// LSPCompletionItemKindEnum indicates an enum completion item.
	LSPCompletionItemKindEnum LSPCompletionItemKind = 13
	// LSPCompletionItemKindKeyword indicates a keyword completion item.
	LSPCompletionItemKindKeyword LSPCompletionItemKind = 14
	// LSPCompletionItemKindSnippet indicates a snippet completion item.
	LSPCompletionItemKindSnippet LSPCompletionItemKind = 15
	// LSPCompletionItemKindColor indicates a color completion item.
	LSPCompletionItemKindColor LSPCompletionItemKind = 16
	// LSPCompletionItemKindFile indicates a file completion item.
	LSPCompletionItemKindFile LSPCompletionItemKind = 17
	// LSPCompletionItemKindReference indicates a reference completion item.
	LSPCompletionItemKindReference LSPCompletionItemKind = 18
	// LSPCompletionItemKindFolder indicates a folder completion item.
	LSPCompletionItemKindFolder LSPCompletionItemKind = 19
	// LSPCompletionItemKindEnumMember indicates an enum member completion item.
	LSPCompletionItemKindEnumMember LSPCompletionItemKind = 20
	// LSPCompletionItemKindConstant indicates a constant completion item.
	LSPCompletionItemKindConstant LSPCompletionItemKind = 21
	// LSPCompletionItemKindStruct indicates a struct completion item.
	LSPCompletionItemKindStruct LSPCompletionItemKind = 22
	// LSPCompletionItemKindEvent indicates an event completion item.
	LSPCompletionItemKindEvent LSPCompletionItemKind = 23
	// LSPCompletionItemKindOperator indicates an operator completion item.
	LSPCompletionItemKindOperator LSPCompletionItemKind = 24
	// LSPCompletionItemKindTypeParameter indicates a type parameter completion item.
	LSPCompletionItemKindTypeParameter LSPCompletionItemKind = 25
)

// LSPInsertTextFormat represents the format of the insert text.
type LSPInsertTextFormat int

const (
	// LSPInsertTextFormatPlainText indicates plain text.
	LSPInsertTextFormatPlainText LSPInsertTextFormat = 1
	// LSPInsertTextFormatSnippet indicates a snippet.
	LSPInsertTextFormatSnippet LSPInsertTextFormat = 2
)

// LSPDebugger provides LSP-compatible debugging output.
// This type converts structured debug events into LSP-compatible diagnostics and other LSP structures.
type LSPDebugger struct {
	debugger    CompilerDebuggerInterface
	fileURI     string
	diagnostics []LSPDiagnostic
	hovers      []LSPHover
	completions []LSPCompletionItem
}

// NewLSPDebugger creates a new LSP-compatible debugger.
func NewLSPDebugger(debugger CompilerDebuggerInterface, fileURI string) *LSPDebugger {
	return &LSPDebugger{
		debugger:    debugger,
		fileURI:     fileURI,
		diagnostics: make([]LSPDiagnostic, 0),
		hovers:      make([]LSPHover, 0),
		completions: make([]LSPCompletionItem, 0),
	}
}

// ConvertDebugEventToDiagnostic converts a debug event to an LSP diagnostic.
func (ld *LSPDebugger) ConvertDebugEventToDiagnostic(event DebugEvent) LSPDiagnostic {
	severity := ld.convertSeverityToLSP(event.Error)

	diagnostic := LSPDiagnostic{
		Range: LSPRange{
			Start: LSPPosition{
				Line:      event.Line - 1, // LSP uses 0-based line numbers
				Character: 0,
			},
			End: LSPPosition{
				Line:      event.Line - 1,
				Character: 100, // Default to end of line
			},
		},
		Severity: severity,
		Source:   string(event.Phase),
		Message:  event.Message,
	}

	if event.Error != nil {
		diagnostic.Code = event.Error.Code
	}

	return diagnostic
}

// convertSeverityToLSP converts internal severity to LSP severity.
func (ld *LSPDebugger) convertSeverityToLSP(errorInfo *ErrorInfo) LSPDiagnosticSeverity {
	if errorInfo == nil {
		return LSPDiagnosticSeverityInformation
	}

	switch errorInfo.Severity {
	case SeverityError:
		return LSPDiagnosticSeverityError
	case SeverityWarning:
		return LSPDiagnosticSeverityWarning
	case SeverityInfo:
		return LSPDiagnosticSeverityInformation
	case SeverityDebug:
		return LSPDiagnosticSeverityHint
	default:
		return LSPDiagnosticSeverityInformation
	}
}

// ConvertDebugEventToHover converts a debug event to an LSP hover.
func (ld *LSPDebugger) ConvertDebugEventToHover(event DebugEvent) LSPHover {
	content := fmt.Sprintf("**%s**\n\n", event.EventType)
	content += fmt.Sprintf("**Phase:** %s\n", event.Phase)
	content += fmt.Sprintf("**Message:** %s\n", event.Message)

	if event.Function != "" {
		content += fmt.Sprintf("**Function:** %s\n", event.Function)
	}

	if event.TypeInfo != nil {
		content += fmt.Sprintf("**Expected Type:** %s\n", event.TypeInfo.ExpectedType)
		content += fmt.Sprintf("**Actual Type:** %s\n", event.TypeInfo.ActualType)
		content += fmt.Sprintf("**Inferred Type:** %s\n", event.TypeInfo.InferredType)
	}

	if event.Scope != nil {
		content += fmt.Sprintf("**Function:** %s\n", event.Scope.FunctionName)
		content += "**Variables:**\n"
		for name, typeName := range event.Scope.Variables {
			content += fmt.Sprintf("  - %s: %s\n", name, typeName)
		}
	}

	if event.Error != nil {
		content += fmt.Sprintf("**Error:** %s\n", event.Error.Message)
		content += "**Suggestions:**\n"
		for _, suggestion := range event.Error.Suggestions {
			content += fmt.Sprintf("  - %s\n", suggestion)
		}
	}

	return LSPHover{
		Contents: LSPMarkedString{
			Language: "markdown",
			Value:    content,
		},
		Range: &LSPRange{
			Start: LSPPosition{
				Line:      event.Line - 1,
				Character: 0,
			},
			End: LSPPosition{
				Line:      event.Line - 1,
				Character: 100,
			},
		},
	}
}

// ConvertDebugEventToCompletion converts a debug event to an LSP completion item.
func (ld *LSPDebugger) ConvertDebugEventToCompletion(event DebugEvent) LSPCompletionItem {
	label := event.EventType
	if event.Function != "" {
		label = event.Function
	}

	detail := event.Message
	if event.TypeInfo != nil && event.TypeInfo.ExpectedType != "" {
		detail = fmt.Sprintf("Expected: %s", event.TypeInfo.ExpectedType)
	}

	documentation := fmt.Sprintf("Phase: %s\nMessage: %s", event.Phase, event.Message)
	if event.Error != nil {
		documentation += fmt.Sprintf("\nError: %s", event.Error.Message)
	}

	return LSPCompletionItem{
		Label:            label,
		Kind:             LSPCompletionItemKindText,
		Detail:           detail,
		Documentation:    documentation,
		InsertText:       label,
		InsertTextFormat: LSPInsertTextFormatPlainText,
	}
}

// ProcessDebugEvents processes all debug events and converts them to LSP structures.
func (ld *LSPDebugger) ProcessDebugEvents() error {
	output, err := ld.debugger.GetAllOutput()
	if err != nil {
		return fmt.Errorf("failed to get debug output: %w", err)
	}

	for _, data := range output {
		var events []DebugEvent
		if err := json.Unmarshal(data, &events); err != nil {
			continue // Skip invalid data
		}

		for _, event := range events {
			// Convert to diagnostic
			diagnostic := ld.ConvertDebugEventToDiagnostic(event)
			ld.diagnostics = append(ld.diagnostics, diagnostic)

			// Convert to hover for type-related events
			if event.TypeInfo != nil || event.Scope != nil {
				hover := ld.ConvertDebugEventToHover(event)
				ld.hovers = append(ld.hovers, hover)
			}

			// Convert to completion for function/type events
			if event.Function != "" || event.EventType == EventFunctionParsed {
				completion := ld.ConvertDebugEventToCompletion(event)
				ld.completions = append(ld.completions, completion)
			}
		}
	}

	return nil
}

// GetDiagnostics returns all LSP diagnostics.
func (ld *LSPDebugger) GetDiagnostics() []LSPDiagnostic {
	return ld.diagnostics
}

// GetHovers returns all LSP hovers.
func (ld *LSPDebugger) GetHovers() []LSPHover {
	return ld.hovers
}

// GetCompletions returns all LSP completion items.
func (ld *LSPDebugger) GetCompletions() []LSPCompletionItem {
	return ld.completions
}

// GetLSPOutput returns all LSP-compatible output as JSON.
func (ld *LSPDebugger) GetLSPOutput() ([]byte, error) {
	output := map[string]interface{}{
		"diagnostics": ld.diagnostics,
		"hovers":      ld.hovers,
		"completions": ld.completions,
		"uri":         ld.fileURI,
		"timestamp":   time.Now(),
	}

	return json.MarshalIndent(output, "", "  ")
}

// LSPNotification represents an LSP notification message.
type LSPNotification struct {
	// JSONRPC is the JSON-RPC version.
	JSONRPC string `json:"jsonrpc"`
	// Method is the method name.
	Method string `json:"method"`
	// Params are the notification parameters.
	Params interface{} `json:"params"`
}

// CreateDiagnosticsNotification creates an LSP diagnostics notification.
func (ld *LSPDebugger) CreateDiagnosticsNotification() LSPNotification {
	return LSPNotification{
		JSONRPC: "2.0",
		Method:  "textDocument/publishDiagnostics",
		Params: map[string]interface{}{
			"uri":         ld.fileURI,
			"diagnostics": ld.diagnostics,
		},
	}
}

// CreateHoverNotification creates an LSP hover notification.
func (ld *LSPDebugger) CreateHoverNotification(position LSPPosition) LSPNotification {
	// Find hover for the given position
	var hover *LSPHover
	for _, h := range ld.hovers {
		if h.Range != nil && ld.positionInRange(position, *h.Range) {
			hover = &h
			break
		}
	}

	return LSPNotification{
		JSONRPC: "2.0",
		Method:  "textDocument/hover",
		Params: map[string]interface{}{
			"position": position,
			"hover":    hover,
		},
	}
}

// positionInRange checks if a position is within a range.
func (ld *LSPDebugger) positionInRange(pos LSPPosition, range_ LSPRange) bool {
	if pos.Line < range_.Start.Line || pos.Line > range_.End.Line {
		return false
	}
	if pos.Line == range_.Start.Line && pos.Character < range_.Start.Character {
		return false
	}
	if pos.Line == range_.End.Line && pos.Character > range_.End.Character {
		return false
	}
	return true
}

// Utility functions for LSP integration

// CreateTypeErrorDiagnostic creates an LSP diagnostic for type errors.
func CreateTypeErrorDiagnostic(fileURI string, line int, expectedType, actualType, context string) LSPDiagnostic {
	return LSPDiagnostic{
		Range: LSPRange{
			Start: LSPPosition{Line: line - 1, Character: 0},
			End:   LSPPosition{Line: line - 1, Character: 100},
		},
		Severity: LSPDiagnosticSeverityError,
		Code:     ErrorCodeTypeMismatch,
		Source:   "forst-typechecker",
		Message:  fmt.Sprintf("Type error in %s: expected %s, got %s", context, expectedType, actualType),
		Tags: []LSPDiagnosticTag{
			LSPDiagnosticTagUnnecessary,
		},
		RelatedInformation: []LSPDiagnosticRelatedInformation{
			{
				Location: LSPLocation{
					URI: fileURI,
					Range: LSPRange{
						Start: LSPPosition{Line: line - 1, Character: 0},
						End:   LSPPosition{Line: line - 1, Character: 100},
					},
				},
				Message: fmt.Sprintf("Expected type: %s", expectedType),
			},
		},
	}
}

// CreateFunctionHover creates an LSP hover for function information.
func CreateFunctionHover(fileURI string, line int, functionName string, parameters, returnTypes []string) LSPHover {
	content := fmt.Sprintf("**Function:** %s\n\n", functionName)
	content += fmt.Sprintf("**Parameters:** %d\n", len(parameters))
	for _, param := range parameters {
		content += fmt.Sprintf("  - %s\n", param)
	}
	content += fmt.Sprintf("**Return Types:** %d\n", len(returnTypes))
	for _, retType := range returnTypes {
		content += fmt.Sprintf("  - %s\n", retType)
	}

	return LSPHover{
		Contents: LSPMarkedString{
			Language: "markdown",
			Value:    content,
		},
		Range: &LSPRange{
			Start: LSPPosition{Line: line - 1, Character: 0},
			End:   LSPPosition{Line: line - 1, Character: 100},
		},
	}
}
