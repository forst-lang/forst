package lsp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestNewLSPServer(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)

	if server == nil {
		t.Fatal("Expected server to be created")
	}

	if server.port != "8080" {
		t.Errorf("Expected port 8080, got %s", server.port)
	}

	if server.log != log {
		t.Error("Expected logger to be set")
	}

	if server.debugger == nil {
		t.Error("Expected debugger to be created")
	}

	if server.lspDebugger == nil {
		t.Error("Expected LSP debugger to be created")
	}
}

func TestHandleHealth(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)

	// Test GET request
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}

	if response["service"] != "forst-lsp" {
		t.Errorf("Expected service 'forst-lsp', got %v", response["service"])
	}

	if response["version"] != "dev" {
		t.Errorf("Expected version 'dev', got %v", response["version"])
	}

	// Test POST request (should fail)
	req, err = http.NewRequest("POST", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	w = httptest.NewRecorder()
	server.handleHealth(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleInitialize(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)

	request := LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{"processId": 123, "rootUri": "file:///tmp", "capabilities": {}}`),
	}

	response := server.handleInitialize(request)

	if response.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC 2.0, got %s", response.JSONRPC)
	}

	if response.ID != 1 {
		t.Errorf("Expected ID 1, got %v", response.ID)
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got %v", response.Error)
	}

	if response.Result == nil {
		t.Fatal("Expected result to be set")
	}

	result, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	capabilities, ok := result["capabilities"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected capabilities to be a map")
	}

	// Check textDocumentSync
	textDocSync, ok := capabilities["textDocumentSync"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected textDocumentSync to be a map")
	}

	if textDocSync["openClose"] != true {
		t.Error("Expected openClose to be true")
	}

	changeValue := textDocSync["change"]
	if changeValue != float64(1) && changeValue != int(1) {
		t.Error("Expected change to be 1, got", changeValue)
	}

	// Check completionProvider
	completionProvider, ok := capabilities["completionProvider"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected completionProvider to be a map")
	}

	triggerChars, ok := completionProvider["triggerCharacters"].([]interface{})
	if !ok {
		// Try alternative type assertion
		if triggerCharsStr, ok := completionProvider["triggerCharacters"].([]string); ok {
			// Convert []string to []interface{} for comparison
			triggerChars = make([]interface{}, len(triggerCharsStr))
			for i, s := range triggerCharsStr {
				triggerChars[i] = s
			}
		} else {
			t.Fatal("Expected triggerCharacters to be a slice")
		}
	}

	expectedChars := []string{".", ":", "("}
	for i, char := range expectedChars {
		if triggerChars[i] != char {
			t.Errorf("Expected trigger character %s, got %v", char, triggerChars[i])
		}
	}

	// Check hoverProvider
	if capabilities["hoverProvider"] != true {
		t.Error("Expected hoverProvider to be true")
	}

	// Check diagnosticProvider
	diagnosticProvider, ok := capabilities["diagnosticProvider"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected diagnosticProvider to be a map")
	}

	if diagnosticProvider["identifier"] != "forst" {
		t.Errorf("Expected identifier 'forst', got %v", diagnosticProvider["identifier"])
	}

	// Check serverInfo
	serverInfo, ok := result["serverInfo"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected serverInfo to be a map")
	}

	if serverInfo["name"] != "forst-lsp" {
		t.Errorf("Expected name 'forst-lsp', got %v", serverInfo["name"])
	}

	if serverInfo["version"] != "dev" {
		t.Errorf("Expected version 'dev', got %v", serverInfo["version"])
	}
}

func TestHandleDidOpen(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)

	// Test with valid Forst file
	request := LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/didOpen",
		Params: json.RawMessage(`{
			"textDocument": {
				"uri": "file:///tmp/test.ft",
				"version": 1,
				"text": "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}"
			}
		}`),
	}

	response := server.handleDidOpen(request)

	if response.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC 2.0, got %s", response.JSONRPC)
	}

	if response.ID != 1 {
		t.Errorf("Expected ID 1, got %v", response.ID)
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got %v", response.Error)
	}

	// Test with invalid JSON
	request.Params = json.RawMessage(`{invalid json`)
	response = server.handleDidOpen(request)

	if response.Error == nil {
		t.Error("Expected error for invalid JSON")
	}

	if response.Error.Code != -32700 {
		t.Errorf("Expected error code -32700, got %d", response.Error.Code)
	}
}

func TestHandleHover(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)

	request := LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/hover",
		Params: json.RawMessage(`{
			"textDocument": {
				"uri": "file:///tmp/test.ft"
			},
			"position": {
				"line": 5,
				"character": 10
			}
		}`),
	}

	response := server.handleHover(request)

	if response.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC 2.0, got %s", response.JSONRPC)
	}

	if response.ID != 1 {
		t.Errorf("Expected ID 1, got %v", response.ID)
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got %v", response.Error)
	}

	if response.Result == nil {
		t.Fatal("Expected result to be set")
	}

	hover, ok := response.Result.(*LSPHover)
	if !ok {
		t.Fatal("Expected result to be LSPHover")
	}

	if hover.Contents.Language != "markdown" {
		t.Errorf("Expected language 'markdown', got %s", hover.Contents.Language)
	}

	if hover.Contents.Value == "" {
		t.Error("Expected hover content to be set")
	}
}

func TestHandleCompletion(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)

	request := LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/completion",
		Params: json.RawMessage(`{
			"textDocument": {
				"uri": "file:///tmp/test.ft"
			},
			"position": {
				"line": 5,
				"character": 10
			}
		}`),
	}

	response := server.handleCompletion(request)

	if response.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC 2.0, got %s", response.JSONRPC)
	}

	if response.ID != 1 {
		t.Errorf("Expected ID 1, got %v", response.ID)
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got %v", response.Error)
	}

	if response.Result == nil {
		t.Fatal("Expected result to be set")
	}

	result, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	if result["isIncomplete"] != false {
		t.Error("Expected isIncomplete to be false")
	}

	items, ok := result["items"].([]LSPCompletionItem)
	if !ok {
		t.Fatal("Expected items to be a slice")
	}

	if len(items) == 0 {
		t.Error("Expected completion items to be provided")
	}

	// Check that we have the expected keywords (based on actual lexer.Keywords)
	expectedKeywords := []string{"func", "type", "var", "const", "if", "else", "for", "return", "ensure", "is", "String", "Int", "Float", "Bool", "Void", "Array", "import", "package", "or", "range", "break", "continue", "switch", "case", "default", "fallthrough", "map", "chan", "interface", "struct", "go", "defer", "goto", "nil"}
	foundKeywords := make(map[string]bool)

	for _, item := range items {
		foundKeywords[item.Label] = true
	}

	for _, keyword := range expectedKeywords {
		if !foundKeywords[keyword] {
			t.Errorf("Expected keyword '%s' to be in completion items", keyword)
		}
	}
}

func TestHandleLSPMethod(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)

	testCases := []struct {
		method      string
		expectError bool
		errorCode   int
		params      json.RawMessage
	}{
		{"initialize", false, 0, json.RawMessage(`{"processId": 123, "rootUri": "file:///tmp", "capabilities": {}}`)},
		{"textDocument/didOpen", false, 0, json.RawMessage(`{"textDocument": {"uri": "file:///tmp/test.ft", "version": 1, "text": "package main"}}`)},
		{"textDocument/didChange", false, 0, json.RawMessage(`{"textDocument": {"uri": "file:///tmp/test.ft", "version": 1}, "contentChanges": [{"text": "package main"}]}`)},
		{"textDocument/didClose", false, 0, json.RawMessage(`{"textDocument": {"uri": "file:///tmp/test.ft"}}`)},
		{"textDocument/publishDiagnostics", false, 0, json.RawMessage(`{}`)},
		{"textDocument/hover", false, 0, json.RawMessage(`{"textDocument": {"uri": "file:///tmp/test.ft"}, "position": {"line": 0, "character": 0}}`)},
		{"textDocument/completion", false, 0, json.RawMessage(`{"textDocument": {"uri": "file:///tmp/test.ft"}, "position": {"line": 0, "character": 0}}`)},
		{"shutdown", false, 0, json.RawMessage(`{}`)},
		{"exit", false, 0, json.RawMessage(`{}`)},
		{"unknown/method", true, -32601, json.RawMessage(`{}`)},
	}

	for _, tc := range testCases {
		t.Run(tc.method, func(t *testing.T) {
			request := LSPRequest{
				JSONRPC: "2.0",
				ID:      1,
				Method:  tc.method,
				Params:  tc.params,
			}

			response := server.handleLSPMethod(request)

			if response.JSONRPC != "2.0" {
				t.Errorf("Expected JSONRPC 2.0, got %s", response.JSONRPC)
			}

			if response.ID != 1 {
				t.Errorf("Expected ID 1, got %v", response.ID)
			}

			if tc.expectError {
				if response.Error == nil {
					t.Error("Expected error")
				} else if response.Error.Code != tc.errorCode {
					t.Errorf("Expected error code %d, got %d", tc.errorCode, response.Error.Code)
				}
			} else {
				if response.Error != nil {
					t.Errorf("Expected no error, got %v", response.Error)
				}
			}
		})
	}
}

func TestHandleLSP(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)

	// Test POST request with valid JSON
	requestBody := `{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "initialize",
		"params": {
			"processId": 123,
			"rootUri": "file:///tmp",
			"capabilities": {}
		}
	}`

	req, err := http.NewRequest("POST", "/", bytes.NewBufferString(requestBody))
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	server.handleLSP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response LSPServerResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}

	if response.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC 2.0, got %s", response.JSONRPC)
	}

	if response.ID != float64(1) {
		t.Errorf("Expected ID 1, got %v", response.ID)
	}

	// Test GET request (should fail)
	req, err = http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	w = httptest.NewRecorder()
	server.handleLSP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}

	// Test POST request with invalid JSON
	req, err = http.NewRequest("POST", "/", bytes.NewBufferString("invalid json"))
	if err != nil {
		t.Fatal(err)
	}

	w = httptest.NewRecorder()
	server.handleLSP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestProcessForstFile(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)

	// Test with .ft file
	diagnostics := server.processForstFile("file:///tmp/test.ft", "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}")

	// Should process .ft files
	if diagnostics == nil {
		t.Error("Expected diagnostics for .ft file")
	}

	// Test with non-.ft file
	diagnostics = server.processForstFile("file:///tmp/test.go", "package main")

	// Should not process non-.ft files
	if diagnostics != nil {
		t.Error("Expected no diagnostics for non-.ft file")
	}
}

func TestFindHoverForPosition(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)

	position := LSPPosition{Line: 5, Character: 10}
	hover := server.findHoverForPosition("file:///tmp/test.ft", position)

	if hover == nil {
		t.Fatal("Expected hover to be created")
	}

	if hover.Contents.Language != "markdown" {
		t.Errorf("Expected language 'markdown', got %s", hover.Contents.Language)
	}

	if hover.Contents.Value == "" {
		t.Error("Expected hover content to be set")
	}

	// Check that the content includes the expected information
	content := hover.Contents.Value
	if len(content) == 0 {
		t.Error("Expected hover content to be non-empty")
	}
}

func TestGetCompletionsForPosition(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)

	position := LSPPosition{Line: 5, Character: 10}
	completions := server.getCompletionsForPosition("file:///tmp/test.ft", position)

	if len(completions) == 0 {
		t.Error("Expected completion items to be provided")
	}

	// Check that we have the expected keywords (based on actual lexer.Keywords)
	expectedKeywords := []string{"func", "type", "var", "const", "if", "else", "for", "return", "ensure", "is", "String", "Int", "Float", "Bool", "Void", "Array", "import", "package", "or", "range", "break", "continue", "switch", "case", "default", "fallthrough", "map", "chan", "interface", "struct", "go", "defer", "goto", "nil"}
	foundKeywords := make(map[string]bool)

	for _, item := range completions {
		foundKeywords[item.Label] = true
	}

	for _, keyword := range expectedKeywords {
		if !foundKeywords[keyword] {
			t.Errorf("Expected keyword '%s' to be in completion items", keyword)
		}
	}

	// Check that each completion item has the expected structure
	for _, item := range completions {
		if item.Label == "" {
			t.Error("Expected completion item to have a label")
		}

		if item.Kind != LSPCompletionItemKindKeyword {
			t.Errorf("Expected completion item kind to be keyword, got %d", item.Kind)
		}

		if item.InsertTextFormat != LSPInsertTextFormatPlainText {
			t.Errorf("Expected insert text format to be plain text, got %d", item.InsertTextFormat)
		}
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)

	// Create a handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Wrap with recovery middleware
	recoveredHandler := server.recoveryMiddleware(panicHandler)

	// Test that panic is recovered
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	recoveredHandler.ServeHTTP(w, req)

	// Should return 500 Internal Server Error
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	if w.Body.String() != "Internal server error\n" {
		t.Errorf("Expected error message, got %s", w.Body.String())
	}
}

func TestCompileForstFileWithPanic(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)

	// Test with content that might cause panic
	diagnostics := server.compileForstFile("/tmp/test.ft", "invalid syntax that might panic", nil)

	// Should not panic and may return nil diagnostics due to panic recovery
	// The important thing is that it doesn't crash the test
	if diagnostics == nil {
		// This is acceptable - panic recovery may return nil
		t.Log("Panic recovery returned nil diagnostics, which is acceptable")
	}
}

func TestHandleDidChange(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)

	request := LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/didChange",
		Params: json.RawMessage(`{
			"textDocument": {
				"uri": "file:///tmp/test.ft",
				"version": 2
			},
			"contentChanges": [
				{
					"text": "package main\n\nfunc main() {\n\tprintln(\"Updated!\")\n}"
				}
			]
		}`),
	}

	response := server.handleDidChange(request)

	if response.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC 2.0, got %s", response.JSONRPC)
	}

	if response.ID != 1 {
		t.Errorf("Expected ID 1, got %v", response.ID)
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got %v", response.Error)
	}
}

func TestHandleDidClose(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)

	request := LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/didClose",
		Params: json.RawMessage(`{
			"textDocument": {
				"uri": "file:///tmp/test.ft"
			}
		}`),
	}

	response := server.handleDidClose(request)

	if response.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC 2.0, got %s", response.JSONRPC)
	}

	if response.ID != 1 {
		t.Errorf("Expected ID 1, got %v", response.ID)
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got %v", response.Error)
	}
}

func TestHandleShutdown(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)

	request := LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "shutdown",
		Params:  json.RawMessage(`{}`),
	}

	response := server.handleShutdown(request)

	if response.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC 2.0, got %s", response.JSONRPC)
	}

	if response.ID != 1 {
		t.Errorf("Expected ID 1, got %v", response.ID)
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got %v", response.Error)
	}

	if response.Result != nil {
		t.Error("Expected result to be nil for shutdown")
	}
}

func TestHandleExit(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)

	request := LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "exit",
		Params:  json.RawMessage(`{}`),
	}

	response := server.handleExit(request)

	if response.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC 2.0, got %s", response.JSONRPC)
	}

	if response.ID != 1 {
		t.Errorf("Expected ID 1, got %v", response.ID)
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got %v", response.Error)
	}

	if response.Result != nil {
		t.Error("Expected result to be nil for exit")
	}
}

func TestSendDiagnosticsNotification(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)

	// Test with empty diagnostics
	server.sendDiagnosticsNotification("file:///tmp/test.ft", []LSPDiagnostic{})

	// Test with some diagnostics
	diagnostics := []LSPDiagnostic{
		{
			Range: LSPRange{
				Start: LSPPosition{Line: 0, Character: 0},
				End:   LSPPosition{Line: 0, Character: 10},
			},
			Severity: LSPDiagnosticSeverityError,
			Message:  "Test error",
		},
	}

	server.sendDiagnosticsNotification("file:///tmp/test.ft", diagnostics)

	// This should not panic and should log the diagnostics
}

func TestProcessForstFileWithNonFtFile(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)

	// Test with .go file (should be ignored)
	diagnostics := server.processForstFile("file:///tmp/test.go", "package main")

	if diagnostics != nil {
		t.Error("Expected no diagnostics for non-.ft file")
	}

	// Test with .ft file
	diagnostics = server.processForstFile("file:///tmp/test.ft", "package main\n\nfunc main() {\n\tprintln(\"Hello\")\n}")

	if diagnostics == nil {
		t.Error("Expected diagnostics for .ft file")
	}
}

func TestFindHoverForPositionWithDifferentPaths(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)

	testCases := []struct {
		uri      string
		position LSPPosition
	}{
		{"file:///tmp/test.ft", LSPPosition{Line: 0, Character: 0}},
		{"file:///home/user/test.ft", LSPPosition{Line: 10, Character: 5}},
		{"file://C:/Users/test.ft", LSPPosition{Line: 5, Character: 15}},
	}

	for _, tc := range testCases {
		t.Run(tc.uri, func(t *testing.T) {
			hover := server.findHoverForPosition(tc.uri, tc.position)

			if hover == nil {
				t.Fatal("Expected hover to be created")
			}

			if hover.Contents.Language != "markdown" {
				t.Errorf("Expected language 'markdown', got %s", hover.Contents.Language)
			}

			if hover.Contents.Value == "" {
				t.Error("Expected hover content to be set")
			}
		})
	}
}

func TestGetCompletionsForPositionWithDifferentPositions(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)

	testCases := []struct {
		uri      string
		position LSPPosition
	}{
		{"file:///tmp/test.ft", LSPPosition{Line: 0, Character: 0}},
		{"file:///home/user/test.ft", LSPPosition{Line: 10, Character: 5}},
		{"file://C:/Users/test.ft", LSPPosition{Line: 5, Character: 15}},
	}

	for _, tc := range testCases {
		t.Run(tc.uri, func(t *testing.T) {
			completions := server.getCompletionsForPosition(tc.uri, tc.position)

			if len(completions) == 0 {
				t.Error("Expected completion items to be provided")
			}

			// Check that we have the expected keywords (based on actual lexer.Keywords)
			expectedKeywords := []string{"func", "type", "var", "const", "if", "else", "for", "return", "ensure", "is", "String", "Int", "Float", "Bool", "Void", "Array", "import", "package", "or", "range", "break", "continue", "switch", "case", "default", "fallthrough", "map", "chan", "interface", "struct", "go", "defer", "goto", "nil"}
			foundKeywords := make(map[string]bool)

			for _, item := range completions {
				foundKeywords[item.Label] = true
			}

			for _, keyword := range expectedKeywords {
				if !foundKeywords[keyword] {
					t.Errorf("Expected keyword '%s' to be in completion items", keyword)
				}
			}
		})
	}
}
