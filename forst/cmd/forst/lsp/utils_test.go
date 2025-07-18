package lsp

import (
	"testing"
	"time"
)

func TestDebugEventBuilder(t *testing.T) {
	// Test basic builder creation
	builder := NewDebugEventBuilder(PhaseLexer, "/test/file.ft")

	if builder == nil {
		t.Fatal("Expected builder to be created")
	}

	// Test fluent interface
	event := builder.
		WithEventType(EventTokenCreated).
		WithMessage("Test message").
		WithLine(10).
		WithFunction("testFunction").
		WithData(map[string]interface{}{"key": "value"}).
		Build()

	if event.Phase != PhaseLexer {
		t.Errorf("Expected phase %s, got %s", PhaseLexer, event.Phase)
	}

	if event.File != "/test/file.ft" {
		t.Errorf("Expected file /test/file.ft, got %s", event.File)
	}

	if event.EventType != EventTokenCreated {
		t.Errorf("Expected event type %s, got %s", EventTokenCreated, event.EventType)
	}

	if event.Message != "Test message" {
		t.Errorf("Expected message 'Test message', got %s", event.Message)
	}

	if event.Line != 10 {
		t.Errorf("Expected line 10, got %d", event.Line)
	}

	if event.Function != "testFunction" {
		t.Errorf("Expected function 'testFunction', got %s", event.Function)
	}

	if event.Data["key"] != "value" {
		t.Errorf("Expected data key 'value', got %v", event.Data["key"])
	}
}

func TestDebugEventBuilderWithScope(t *testing.T) {
	builder := NewDebugEventBuilder(PhaseTypechecker, "/test/file.ft")

	scopeInfo := NewScopeInfoBuilder().
		WithFunctionName("testFunc").
		WithVariable("x", "Int").
		WithType("MyType", "struct").
		WithStackLevel("function").
		Build()

	event := builder.
		WithEventType(EventScopeEntered).
		WithScope(scopeInfo).
		Build()

	if event.Scope == nil {
		t.Fatal("Expected scope to be set")
	}

	if event.Scope.FunctionName != "testFunc" {
		t.Errorf("Expected function name 'testFunc', got %s", event.Scope.FunctionName)
	}

	if event.Scope.Variables["x"] != "Int" {
		t.Errorf("Expected variable x to be Int, got %s", event.Scope.Variables["x"])
	}

	if event.Scope.Types["MyType"] != "struct" {
		t.Errorf("Expected type MyType to be struct, got %s", event.Scope.Types["MyType"])
	}

	if len(event.Scope.Stack) != 1 || event.Scope.Stack[0] != "function" {
		t.Errorf("Expected stack to contain 'function', got %v", event.Scope.Stack)
	}
}

func TestDebugEventBuilderWithError(t *testing.T) {
	builder := NewDebugEventBuilder(PhaseParser, "/test/file.ft")

	errorInfo := NewErrorInfoBuilder(ErrorCodeUnexpectedToken, "Unexpected token").
		WithSeverity(SeverityError).
		WithSuggestion("Check syntax").
		WithSuggestion("Verify token order").
		Build()

	event := builder.
		WithEventType(EventParserError).
		WithError(errorInfo).
		Build()

	if event.Error == nil {
		t.Fatal("Expected error to be set")
	}

	if event.Error.Code != ErrorCodeUnexpectedToken {
		t.Errorf("Expected error code %s, got %s", ErrorCodeUnexpectedToken, event.Error.Code)
	}

	if event.Error.Message != "Unexpected token" {
		t.Errorf("Expected error message 'Unexpected token', got %s", event.Error.Message)
	}

	if event.Error.Severity != SeverityError {
		t.Errorf("Expected severity %s, got %s", SeverityError, event.Error.Severity)
	}

	if len(event.Error.Suggestions) != 2 {
		t.Errorf("Expected 2 suggestions, got %d", len(event.Error.Suggestions))
	}

	expectedSuggestions := []string{"Check syntax", "Verify token order"}
	for i, suggestion := range expectedSuggestions {
		if event.Error.Suggestions[i] != suggestion {
			t.Errorf("Expected suggestion %s, got %s", suggestion, event.Error.Suggestions[i])
		}
	}
}

func TestScopeInfoBuilder(t *testing.T) {
	builder := NewScopeInfoBuilder()

	scope := builder.
		WithFunctionName("testFunction").
		WithVariable("x", "Int").
		WithVariable("y", "String").
		WithType("MyStruct", "struct").
		WithStackLevel("global").
		WithStackLevel("function").
		Build()

	if scope == nil {
		t.Fatal("Expected scope to be created")
	}

	if scope.FunctionName != "testFunction" {
		t.Errorf("Expected function name 'testFunction', got %s", scope.FunctionName)
	}

	if len(scope.Variables) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(scope.Variables))
	}

	if scope.Variables["x"] != "Int" {
		t.Errorf("Expected variable x to be Int, got %s", scope.Variables["x"])
	}

	if scope.Variables["y"] != "String" {
		t.Errorf("Expected variable y to be String, got %s", scope.Variables["y"])
	}

	if len(scope.Types) != 1 {
		t.Errorf("Expected 1 type, got %d", len(scope.Types))
	}

	if scope.Types["MyStruct"] != "struct" {
		t.Errorf("Expected type MyStruct to be struct, got %s", scope.Types["MyStruct"])
	}

	if len(scope.Stack) != 2 {
		t.Errorf("Expected 2 stack levels, got %d", len(scope.Stack))
	}

	expectedStack := []string{"global", "function"}
	for i, level := range expectedStack {
		if scope.Stack[i] != level {
			t.Errorf("Expected stack level %s, got %s", level, scope.Stack[i])
		}
	}
}

func TestErrorInfoBuilder(t *testing.T) {
	builder := NewErrorInfoBuilder(ErrorCodeTypeMismatch, "Type mismatch error")

	errorInfo := builder.
		WithSeverity(SeverityWarning).
		WithSuggestion("Check type annotations").
		WithSuggestion("Verify variable declarations").
		Build()

	if errorInfo == nil {
		t.Fatal("Expected error info to be created")
	}

	if errorInfo.Code != ErrorCodeTypeMismatch {
		t.Errorf("Expected error code %s, got %s", ErrorCodeTypeMismatch, errorInfo.Code)
	}

	if errorInfo.Message != "Type mismatch error" {
		t.Errorf("Expected error message 'Type mismatch error', got %s", errorInfo.Message)
	}

	if errorInfo.Severity != SeverityWarning {
		t.Errorf("Expected severity %s, got %s", SeverityWarning, errorInfo.Severity)
	}

	if len(errorInfo.Suggestions) != 2 {
		t.Errorf("Expected 2 suggestions, got %d", len(errorInfo.Suggestions))
	}

	expectedSuggestions := []string{"Check type annotations", "Verify variable declarations"}
	for i, suggestion := range expectedSuggestions {
		if errorInfo.Suggestions[i] != suggestion {
			t.Errorf("Expected suggestion %s, got %s", suggestion, errorInfo.Suggestions[i])
		}
	}
}

func TestCreateTokenEvent(t *testing.T) {
	event := CreateTokenEvent(PhaseLexer, "/test/file.ft", "IDENTIFIER", "myVar", 5, 10)

	if event.Phase != PhaseLexer {
		t.Errorf("Expected phase %s, got %s", PhaseLexer, event.Phase)
	}

	if event.File != "/test/file.ft" {
		t.Errorf("Expected file /test/file.ft, got %s", event.File)
	}

	if event.EventType != EventTokenCreated {
		t.Errorf("Expected event type %s, got %s", EventTokenCreated, event.EventType)
	}

	if event.Message != "Created IDENTIFIER token: myVar" {
		t.Errorf("Expected message 'Created IDENTIFIER token: myVar', got %s", event.Message)
	}

	if event.Line != 5 {
		t.Errorf("Expected line 5, got %d", event.Line)
	}

	if event.Data["token_type"] != "IDENTIFIER" {
		t.Errorf("Expected token_type IDENTIFIER, got %v", event.Data["token_type"])
	}

	if event.Data["value"] != "myVar" {
		t.Errorf("Expected value myVar, got %v", event.Data["value"])
	}

	if event.Data["line"] != 5 {
		t.Errorf("Expected line 5, got %v", event.Data["line"])
	}

	if event.Data["column"] != 10 {
		t.Errorf("Expected column 10, got %v", event.Data["column"])
	}
}

func TestCreateFunctionEvent(t *testing.T) {
	parameters := []string{"Int", "String"}
	returnTypes := []string{"Bool"}

	event := CreateFunctionEvent(PhaseParser, "/test/file.ft", "testFunction", parameters, returnTypes)

	if event.Phase != PhaseParser {
		t.Errorf("Expected phase %s, got %s", PhaseParser, event.Phase)
	}

	if event.File != "/test/file.ft" {
		t.Errorf("Expected file /test/file.ft, got %s", event.File)
	}

	if event.EventType != EventFunctionParsed {
		t.Errorf("Expected event type %s, got %s", EventFunctionParsed, event.EventType)
	}

	if event.Message != "Processed function: testFunction" {
		t.Errorf("Expected message 'Processed function: testFunction', got %s", event.Message)
	}

	if event.Function != "testFunction" {
		t.Errorf("Expected function 'testFunction', got %s", event.Function)
	}

	if event.Data["function_name"] != "testFunction" {
		t.Errorf("Expected function_name testFunction, got %v", event.Data["function_name"])
	}

	params := event.Data["parameters"].([]string)
	if len(params) != 2 || params[0] != "Int" || params[1] != "String" {
		t.Errorf("Expected parameters [Int String], got %v", params)
	}

	returns := event.Data["return_types"].([]string)
	if len(returns) != 1 || returns[0] != "Bool" {
		t.Errorf("Expected return_types [Bool], got %v", returns)
	}
}

func TestCreateTypeErrorEvent(t *testing.T) {
	event := CreateTypeErrorEvent(PhaseTypechecker, "/test/file.ft", "Int", "String", "variable assignment")

	if event.Phase != PhaseTypechecker {
		t.Errorf("Expected phase %s, got %s", PhaseTypechecker, event.Phase)
	}

	if event.File != "/test/file.ft" {
		t.Errorf("Expected file /test/file.ft, got %s", event.File)
	}

	if event.EventType != EventTypeError {
		t.Errorf("Expected event type %s, got %s", EventTypeError, event.EventType)
	}

	expectedMessage := "Type error in variable assignment: expected Int, got String"
	if event.Message != expectedMessage {
		t.Errorf("Expected message '%s', got %s", expectedMessage, event.Message)
	}

	if event.Error == nil {
		t.Fatal("Expected error to be set")
	}

	if event.Error.Code != ErrorCodeTypeMismatch {
		t.Errorf("Expected error code %s, got %s", ErrorCodeTypeMismatch, event.Error.Code)
	}

	expectedErrorMsg := "Type mismatch: expected Int, got String"
	if event.Error.Message != expectedErrorMsg {
		t.Errorf("Expected error message '%s', got %s", expectedErrorMsg, event.Error.Message)
	}

	if event.Error.Severity != SeverityError {
		t.Errorf("Expected severity %s, got %s", SeverityError, event.Error.Severity)
	}

	if len(event.Error.Suggestions) != 2 {
		t.Errorf("Expected 2 suggestions, got %d", len(event.Error.Suggestions))
	}

	if event.TypeInfo == nil {
		t.Fatal("Expected type info to be set")
	}

	if event.TypeInfo.ExpectedType != "Int" {
		t.Errorf("Expected expected type Int, got %s", event.TypeInfo.ExpectedType)
	}

	if event.TypeInfo.ActualType != "String" {
		t.Errorf("Expected actual type String, got %s", event.TypeInfo.ActualType)
	}

	if len(event.TypeInfo.TypeErrors) != 1 {
		t.Errorf("Expected 1 type error, got %d", len(event.TypeInfo.TypeErrors))
	}

	expectedTypeError := "Expected Int, got String"
	if event.TypeInfo.TypeErrors[0] != expectedTypeError {
		t.Errorf("Expected type error '%s', got %s", expectedTypeError, event.TypeInfo.TypeErrors[0])
	}
}

func TestCreateScopeEvent(t *testing.T) {
	variables := map[string]string{
		"x": "Int",
		"y": "String",
	}

	event := CreateScopeEvent(PhaseTypechecker, "/test/file.ft", EventScopeEntered, "testFunction", variables)

	if event.Phase != PhaseTypechecker {
		t.Errorf("Expected phase %s, got %s", PhaseTypechecker, event.Phase)
	}

	if event.File != "/test/file.ft" {
		t.Errorf("Expected file /test/file.ft, got %s", event.File)
	}

	if event.EventType != EventScopeEntered {
		t.Errorf("Expected event type %s, got %s", EventScopeEntered, event.EventType)
	}

	expectedMessage := "Scope scope_entered for function: testFunction"
	if event.Message != expectedMessage {
		t.Errorf("Expected message '%s', got %s", expectedMessage, event.Message)
	}

	if event.Function != "testFunction" {
		t.Errorf("Expected function 'testFunction', got %s", event.Function)
	}

	if event.Scope == nil {
		t.Fatal("Expected scope to be set")
	}

	if event.Scope.FunctionName != "testFunction" {
		t.Errorf("Expected function name 'testFunction', got %s", event.Scope.FunctionName)
	}

	if len(event.Scope.Variables) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(event.Scope.Variables))
	}

	if event.Scope.Variables["x"] != "Int" {
		t.Errorf("Expected variable x to be Int, got %s", event.Scope.Variables["x"])
	}

	if event.Scope.Variables["y"] != "String" {
		t.Errorf("Expected variable y to be String, got %s", event.Scope.Variables["y"])
	}
}

func TestFormatDebugEvent(t *testing.T) {
	event := DebugEvent{
		Timestamp: time.Now(),
		Phase:     PhaseLexer,
		File:      "/test/file.ft",
		EventType: EventTokenCreated,
		Message:   "Created token",
		Line:      10,
		Function:  "testFunction",
	}

	formatted := FormatDebugEvent(event)

	expectedParts := []string{
		"[lexer]",
		"[token_created]",
		"file=/test/file.ft",
		"line=10",
		"function=testFunction",
		"Created token",
	}

	for _, part := range expectedParts {
		if !contains(formatted, part) {
			t.Errorf("Expected formatted string to contain '%s', got '%s'", part, formatted)
		}
	}
}

func TestFormatDebugEventMinimal(t *testing.T) {
	event := DebugEvent{
		Timestamp: time.Now(),
		Phase:     PhaseParser,
		EventType: EventParserComplete,
		Message:   "Parser complete",
	}

	formatted := FormatDebugEvent(event)

	expectedParts := []string{
		"[parser]",
		"[parser_complete]",
		"Parser complete",
	}

	for _, part := range expectedParts {
		if !contains(formatted, part) {
			t.Errorf("Expected formatted string to contain '%s', got '%s'", part, formatted)
		}
	}
}

func TestValidateEventType(t *testing.T) {
	testCases := []struct {
		name      string
		eventType string
		expectErr bool
	}{
		{"valid lexer event", EventTokenCreated, false},
		{"valid parser event", EventFunctionParsed, false},
		{"valid typechecker event", EventTypeInferred, false},
		{"valid transformer event", EventFunctionTransformed, false},
		{"valid discovery event", EventFunctionDiscovered, false},
		{"valid executor event", EventFunctionExecuted, false},
		{"invalid event", "invalid_event", true},
		{"empty event", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateEventType(tc.eventType)

			if tc.expectErr {
				if err == nil {
					t.Error("Expected error for invalid event type")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for valid event type, got %v", err)
				}
			}
		})
	}
}

func TestValidateErrorCode(t *testing.T) {
	testCases := []struct {
		name      string
		errorCode string
		expectErr bool
	}{
		{"valid lexer error", ErrorCodeInvalidToken, false},
		{"valid parser error", ErrorCodeUnexpectedToken, false},
		{"valid typechecker error", ErrorCodeTypeMismatch, false},
		{"valid transformer error", ErrorCodeTransformationFailed, false},
		{"invalid error code", "INVALID_ERROR_CODE", true},
		{"empty error code", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateErrorCode(tc.errorCode)

			if tc.expectErr {
				if err == nil {
					t.Error("Expected error for invalid error code")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for valid error code, got %v", err)
				}
			}
		})
	}
}

func TestDebugEventBuilderChaining(t *testing.T) {
	// Test that all builder methods can be chained
	event := NewDebugEventBuilder(PhaseTransformer, "/test/file.ft").
		WithEventType(EventFunctionTransformed).
		WithMessage("Function transformed").
		WithLine(15).
		WithFunction("myFunction").
		WithData(map[string]interface{}{"transformed": true}).
		WithScope(NewScopeInfoBuilder().WithFunctionName("myFunction").Build()).
		WithAST(&ASTInfo{NodeType: "FunctionNode"}).
		WithTypeInfo(&TypeInfo{ExpectedType: "Int"}).
		WithError(NewErrorInfoBuilder(ErrorCodeTransformationFailed, "Transformation failed").Build()).
		Build()

	if event.Phase != PhaseTransformer {
		t.Errorf("Expected phase %s, got %s", PhaseTransformer, event.Phase)
	}

	if event.EventType != EventFunctionTransformed {
		t.Errorf("Expected event type %s, got %s", EventFunctionTransformed, event.EventType)
	}

	if event.Message != "Function transformed" {
		t.Errorf("Expected message 'Function transformed', got %s", event.Message)
	}

	if event.Line != 15 {
		t.Errorf("Expected line 15, got %d", event.Line)
	}

	if event.Function != "myFunction" {
		t.Errorf("Expected function 'myFunction', got %s", event.Function)
	}

	if event.Data["transformed"] != true {
		t.Errorf("Expected data transformed=true, got %v", event.Data["transformed"])
	}

	if event.Scope == nil {
		t.Error("Expected scope to be set")
	}

	if event.AST == nil {
		t.Error("Expected AST to be set")
	}

	if event.TypeInfo == nil {
		t.Error("Expected type info to be set")
	}

	if event.Error == nil {
		t.Error("Expected error to be set")
	}
}

func TestScopeInfoBuilderChaining(t *testing.T) {
	// Test that all scope builder methods can be chained
	scope := NewScopeInfoBuilder().
		WithFunctionName("testFunc").
		WithVariable("x", "Int").
		WithVariable("y", "String").
		WithType("MyType", "struct").
		WithStackLevel("global").
		WithStackLevel("function").
		WithStackLevel("block").
		Build()

	if scope.FunctionName != "testFunc" {
		t.Errorf("Expected function name 'testFunc', got %s", scope.FunctionName)
	}

	if len(scope.Variables) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(scope.Variables))
	}

	if len(scope.Types) != 1 {
		t.Errorf("Expected 1 type, got %d", len(scope.Types))
	}

	if len(scope.Stack) != 3 {
		t.Errorf("Expected 3 stack levels, got %d", len(scope.Stack))
	}
}

func TestErrorInfoBuilderChaining(t *testing.T) {
	// Test that all error builder methods can be chained
	errorInfo := NewErrorInfoBuilder(ErrorCodeUndefinedVariable, "Variable not found").
		WithSeverity(SeverityWarning).
		WithSuggestion("Declare the variable").
		WithSuggestion("Check variable scope").
		WithSuggestion("Import required module").
		Build()

	if errorInfo.Code != ErrorCodeUndefinedVariable {
		t.Errorf("Expected error code %s, got %s", ErrorCodeUndefinedVariable, errorInfo.Code)
	}

	if errorInfo.Message != "Variable not found" {
		t.Errorf("Expected error message 'Variable not found', got %s", errorInfo.Message)
	}

	if errorInfo.Severity != SeverityWarning {
		t.Errorf("Expected severity %s, got %s", SeverityWarning, errorInfo.Severity)
	}

	if len(errorInfo.Suggestions) != 3 {
		t.Errorf("Expected 3 suggestions, got %d", len(errorInfo.Suggestions))
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
