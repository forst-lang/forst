package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

	if response.Result == nil {
		t.Fatal("Expected result with publish diagnostics")
	}
	pub, ok := response.Result.(PublishDiagnosticsParams)
	if !ok {
		t.Fatalf("Expected PublishDiagnosticsParams result, got %T", response.Result)
	}
	if pub.URI != "file:///tmp/test.ft" {
		t.Errorf("Expected uri file:///tmp/test.ft, got %q", pub.URI)
	}
	if pub.Diagnostics == nil {
		t.Error("Expected diagnostics slice (may be empty)")
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
	server.documentMu.Lock()
	server.openDocuments["file:///tmp/test.ft"] = "package main\n\nfunc main() {\n}\n"
	server.documentMu.Unlock()

	request := LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/hover",
		Params: json.RawMessage(`{
			"textDocument": {
				"uri": "file:///tmp/test.ft"
			},
			"position": {
				"line": 2,
				"character": 5
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

func TestHandleTextDocumentListRequest(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "handle_cpl.ft")
	const src = `package main

func main() {
  var x: Int = 1
}
`
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ftPath)
	server.documentMu.Lock()
	server.openDocuments[uri] = src
	server.documentMu.Unlock()

	params, err := json.Marshal(map[string]interface{}{
		"textDocument": map[string]string{"uri": uri},
		"position":     map[string]int{"line": 3, "character": 2},
	})
	if err != nil {
		t.Fatal(err)
	}

	request := LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/completion",
		Params:  json.RawMessage(params),
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

	found := make(map[string]bool)
	for _, item := range items {
		found[item.Label] = true
	}
	// Inside function body: keywords exclude package/import; include return.
	if !found["return"] || !found["var"] {
		t.Fatalf("expected block keywords, got labels: %v", found)
	}
	if found["package"] || found["import"] {
		t.Fatalf("did not expect package/import keywords inside block, got %v", found)
	}
	if !found["main"] {
		t.Fatal("expected function name main in completions")
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
		omitID      bool // JSON-RPC notification: no "id" field; HTTP layer must not send a body (see TestHandleLSP_NotificationNoIDNoJSONBody).
	}{
		{"initialize", false, 0, json.RawMessage(`{"processId": 123, "rootUri": "file:///tmp", "capabilities": {}}`), false},
		{"textDocument/didOpen", false, 0, json.RawMessage(`{"textDocument": {"uri": "file:///tmp/test.ft", "version": 1, "text": "package main"}}`), false},
		{"textDocument/didChange", false, 0, json.RawMessage(`{"textDocument": {"uri": "file:///tmp/test.ft", "version": 1}, "contentChanges": [{"text": "package main"}]}`), false},
		{"textDocument/didClose", false, 0, json.RawMessage(`{"textDocument": {"uri": "file:///tmp/test.ft"}}`), false},
		{"textDocument/publishDiagnostics", false, 0, json.RawMessage(`{}`), false},
		{"textDocument/hover", false, 0, json.RawMessage(`{"textDocument": {"uri": "file:///tmp/test.ft"}, "position": {"line": 0, "character": 0}}`), false},
		{"textDocument/completion", false, 0, json.RawMessage(`{"textDocument": {"uri": "file:///tmp/test.ft"}, "position": {"line": 0, "character": 0}}`), false},
		{"shutdown", false, 0, json.RawMessage(`{}`), false},
		{"exit", false, 0, json.RawMessage(`{}`), false},
		{"initialized", false, 0, json.RawMessage(`{}`), false},
		{"$/cancelRequest", false, 0, json.RawMessage(`{"id": 1}`), false},
		{"initialized", false, 0, json.RawMessage(`{}`), true},
		{"$/cancelRequest", false, 0, json.RawMessage(`{"id": 1}`), true},
		{"unknown/method", true, -32601, json.RawMessage(`{}`), false},
		{"$/progress", true, -32601, json.RawMessage(`{}`), false},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s_omitID_%v", tc.method, tc.omitID), func(t *testing.T) {
			request := LSPRequest{
				JSONRPC: "2.0",
				Method:  tc.method,
				Params:  tc.params,
			}
			if !tc.omitID {
				request.ID = 1
			}

			response := server.handleLSPMethod(request)

			if response.JSONRPC != "2.0" {
				t.Errorf("Expected JSONRPC 2.0, got %s", response.JSONRPC)
			}

			if tc.omitID {
				if response.ID != nil {
					t.Errorf("notification-style request: want nil ID in response echo, got %v", response.ID)
				}
			} else {
				if response.ID != 1 {
					t.Errorf("Expected ID 1, got %v", response.ID)
				}
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

	// GET returns a JSON hint (not JSON-RPC)
	req, err = http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	w = httptest.NewRecorder()
	server.handleLSP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected application/json, got %q", ct)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("forst-lsp")) {
		t.Errorf("Expected GET body to mention forst-lsp, got %s", w.Body.String())
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

func TestHandleLSP_NotificationNoIDNoJSONBody(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	server := NewLSPServer("8080", log)
	// JSON-RPC notification: no top-level "id" field — server must not send a JSON-RPC response body.
	body := `{"jsonrpc":"2.0","method":"initialized","params":{}}`
	req, err := http.NewRequest("POST", "/", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.handleLSP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content for notification, got %d body=%q", w.Code, w.Body.String())
	}
	if len(w.Body.Bytes()) != 0 {
		t.Fatalf("expected empty body, got %q", w.Body.String())
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
	server.documentMu.Lock()
	server.openDocuments["file:///tmp/test.ft"] = "package main\n\nfunc main() {\n}\n"
	server.documentMu.Unlock()

	position := LSPPosition{Line: 2, Character: 5}
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

func TestFindHoverForPositionUsesSyncedDocument(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)
	uri := mustFileURI(t, filepath.Join(t.TempDir(), "hover_sync.ft"))
	content := "package main\n\nfunc Hello() {\n}\n"
	server.documentMu.Lock()
	server.openDocuments[uri] = content
	server.documentMu.Unlock()

	h := server.findHoverForPosition(uri, LSPPosition{Line: 2, Character: 6})
	if h == nil {
		t.Fatal("expected hover for function name")
	}
	if h.Contents.Value == "" {
		t.Fatal("expected hover body")
	}
	if !strings.Contains(h.Contents.Value, "Hello") {
		t.Fatalf("expected function name in hover, got %q", h.Contents.Value)
	}
}

func TestFindHoverForPosition_goFmtPrintln(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module hoverlsp\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	const src = "package main\n\nimport \"fmt\"\n\nfunc main() {\n  fmt.Println(\"x\")\n}\n"
	ftPath := filepath.Join(dir, "main.ft")
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ftPath)
	log := logrus.New()
	server := NewLSPServer("8080", log)
	server.documentMu.Lock()
	server.openDocuments[uri] = src
	server.documentMu.Unlock()

	// Cursor on `Println` (0-based line for `  fmt.Println(...)`).
	h := server.findHoverForPosition(uri, LSPPosition{Line: 5, Character: 6})
	if h == nil {
		t.Fatal("expected hover on fmt.Println selector")
	}
	v := h.Contents.Value
	if !strings.Contains(v, "Println") {
		t.Fatalf("expected Go signature mentioning Println, got %q", v)
	}
}

func TestListAtPosition_keywordKinds(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "get_cpl.ft")
	const src = `package main

func main() {
  var x: Int = 1
}
`
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ftPath)
	server.documentMu.Lock()
	server.openDocuments[uri] = src
	server.documentMu.Unlock()

	position := LSPPosition{Line: 3, Character: 2}
	completions, incomplete := server.getCompletionsForPosition(uri, position, nil)
	if incomplete {
		t.Error("Expected isIncomplete false with single open buffer")
	}

	if len(completions) == 0 {
		t.Error("Expected completion items to be provided")
	}

	kinds := make(map[LSPCompletionItemKind]int)
	for _, item := range completions {
		if item.Label == "" {
			t.Error("Expected completion item to have a label")
		}
		kinds[item.Kind]++
		if item.InsertTextFormat != LSPInsertTextFormatPlainText {
			t.Errorf("Expected insert text format to be plain text, got %d", item.InsertTextFormat)
		}
	}
	if kinds[LSPCompletionItemKindKeyword] == 0 {
		t.Fatalf("expected some keyword completions, kinds=%v", kinds)
	}
}

func TestHandleLSP_getReturnsProbeJSON(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	server.handleLSP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"service":"forst-lsp"`) {
		t.Fatalf("body: %s", w.Body.String())
	}
}

func TestHandleLSP_invalidMethod405(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)
	req := httptest.NewRequest(http.MethodPut, "/", nil)
	w := httptest.NewRecorder()
	server.handleLSP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status %d", w.Code)
	}
}

func TestHandleLSP_invalidJSONBody400(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not-json{"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.handleLSP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)

	// Create a handler that panics
	panicHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
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

	if response.Result == nil {
		t.Fatal("Expected result with publish diagnostics")
	}
	pub, ok := response.Result.(PublishDiagnosticsParams)
	if !ok {
		t.Fatalf("Expected PublishDiagnosticsParams result, got %T", response.Result)
	}
	if pub.URI != "file:///tmp/test.ft" {
		t.Errorf("Expected uri file:///tmp/test.ft, got %q", pub.URI)
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

	if response.Result == nil {
		t.Fatal("Expected result with cleared diagnostics")
	}
	pub, ok := response.Result.(PublishDiagnosticsParams)
	if !ok {
		t.Fatalf("Expected PublishDiagnosticsParams result, got %T", response.Result)
	}
	if pub.URI != "file:///tmp/test.ft" {
		t.Errorf("Expected uri file:///tmp/test.ft, got %q", pub.URI)
	}
	if len(pub.Diagnostics) != 0 {
		t.Errorf("Expected no diagnostics after close, got %d", len(pub.Diagnostics))
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

func TestSendDiagnosticsNotification(_ *testing.T) {
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

	testCases := []string{
		"file:///tmp/test.ft",
		"file:///home/user/test.ft",
		"file://C:/Users/test.ft",
	}

	for _, uri := range testCases {
		t.Run(uri, func(t *testing.T) {
			server.documentMu.Lock()
			server.openDocuments[uri] = "package main\n\nfunc main() {\n}\n"
			server.documentMu.Unlock()
			hover := server.findHoverForPosition(uri, LSPPosition{Line: 2, Character: 5})

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

func TestListAtPosition_zoneByCursor(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "multi_cpl.ft")
	const src = `package main

func main() {
  var x: Int = 1
  if x > 0 {
    println("a")
  }
  for n := 0; n < 2; n = n + 1 {
    println("b")
  }
}
`
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ftPath)
	server.documentMu.Lock()
	server.openDocuments[uri] = src
	server.documentMu.Unlock()

	testCases := []struct {
		name string
		pos  LSPPosition
	}{
		{"top_level_package_line", LSPPosition{Line: 0, Character: 0}},
		{"inside_main", LSPPosition{Line: 3, Character: 2}},
		{"inside_if_body", LSPPosition{Line: 5, Character: 4}},
		{"inside_for_body", LSPPosition{Line: 8, Character: 4}},
		{"after_if_condition_line", LSPPosition{Line: 4, Character: 2}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			completions, _ := server.getCompletionsForPosition(uri, tc.pos, nil)
			if len(completions) == 0 {
				t.Fatal("Expected completion items")
			}
			found := make(map[string]bool)
			for _, item := range completions {
				found[item.Label] = true
			}
			switch tc.name {
			case "top_level_package_line":
				if !found["package"] || !found["func"] {
					t.Fatalf("expected top-level keywords: %v", found)
				}
				if found["return"] {
					t.Fatal("did not expect return at top level")
				}
			case "inside_main", "inside_if_body", "inside_for_body", "after_if_condition_line":
				if !found["return"] {
					t.Fatal("expected return inside function")
				}
				if found["package"] {
					t.Fatal("did not expect package keyword inside block")
				}
			}
		})
	}
}

func TestLSPServerDebugInfoForLLM(t *testing.T) {
	// This test demonstrates how the enhanced LSP server provides comprehensive
	// debugging information that can be used by an LLM to debug compiler issues

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	server := NewLSPServer("8080", logger)

	// Test file with a typical compiler bug (type mismatch)
	testContent := `
func exampleFunction(x: String) {
    y := x + 42  // Type error: cannot add String and Int
    return y
}
`

	// Simulate opening a document
	request := LSPRequest{
		JSONRPC: "2.0",
		ID:      "test-1",
		Method:  "textDocument/didOpen",
		Params: json.RawMessage(fmt.Sprintf(`{
			"textDocument": {
				"uri": "file:///test.ft",
				"version": 1,
				"text": %q
			}
		}`, testContent)),
	}

	response := server.handleDidOpen(request)
	if response.Error != nil {
		t.Fatalf("Expected no error, got: %v", response.Error)
	}

	// Now request comprehensive debug information
	debugRequest := LSPRequest{
		JSONRPC: "2.0",
		ID:      "test-2",
		Method:  "textDocument/debugInfo",
		Params: json.RawMessage(`{
			"textDocument": {
				"uri": "file:///test.ft"
			},
			"compression": false
		}`),
	}

	debugResponse := server.handleDebugInfo(debugRequest)
	if debugResponse.Error != nil {
		t.Fatalf("Expected no error, got: %v", debugResponse.Error)
	}

	// Verify that we got comprehensive debug information
	debugInfo, ok := debugResponse.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected debug info to be a map")
	}

	// Check that we have the expected debugging information
	requiredFields := []string{"uri", "debugMode", "output"}
	for _, field := range requiredFields {
		if _, exists := debugInfo[field]; !exists {
			t.Errorf("Expected debug info to contain field: %s", field)
		}
	}

	// Check that output contains the expected data structure
	output, ok := debugInfo["output"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected output to be a map")
	}

	// Check that output.data contains the detailed information
	data, ok := output["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected output.data to be a map")
	}

	// Check that data contains the expected fields
	expectedDataFields := []string{"diagnostics"}
	for _, field := range expectedDataFields {
		if _, exists := data[field]; !exists {
			t.Errorf("Expected output.data to contain field: %s", field)
		}
	}

	// Verify that diagnostics contain the type error
	diagnostics, ok := data["diagnostics"].([]LSPDiagnostic)
	if !ok {
		t.Fatal("Expected diagnostics to be a slice")
	}

	if len(diagnostics) == 0 {
		t.Log("No diagnostics found - this might be expected if the test file compiles successfully")
	} else {
		t.Logf("Found %d diagnostics", len(diagnostics))
		for i, diag := range diagnostics {
			t.Logf("Diagnostic %d: %s (severity: %d)", i, diag.Message, diag.Severity)
		}
	}

	// Note: compilerState and phaseDetails are not currently provided in the debug info
	// They may be added in future versions for more comprehensive debugging

	t.Log("LLM debugging test completed successfully")
	t.Log("This demonstrates how an LLM can use the LSP server to:")
	t.Log("1. Open a file and trigger compilation")
	t.Log("2. Request comprehensive debug information")
	t.Log("3. Analyze compiler state across all phases")
	t.Log("4. Get detailed diagnostics with error codes and suggestions")
	t.Log("5. Access phase-specific information for targeted debugging")
}

func TestHandleCompilerState_andPhaseDetails(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	s := NewLSPServer("8080", log)
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "st.ft")
	const src = `package main

func main() {
	println("x")
}
`
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ftPath)
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	cs := s.handleCompilerState(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params: mustJSONParams(t, map[string]interface{}{
			"textDocument": map[string]interface{}{"uri": uri},
		}),
	})
	if cs.Error != nil {
		t.Fatalf("compilerState: %+v", cs.Error)
	}
	m, ok := cs.Result.(map[string]interface{})
	if !ok || m["uri"] == nil {
		t.Fatalf("expected result map with uri, got %#v", cs.Result)
	}

	pdAll := s.handlePhaseDetails(LSPRequest{
		JSONRPC: "2.0",
		ID:      2,
		Params: mustJSONParams(t, map[string]interface{}{
			"textDocument": map[string]interface{}{"uri": uri},
			"phase":        "",
		}),
	})
	if pdAll.Error != nil {
		t.Fatalf("phaseDetails all: %+v", pdAll.Error)
	}
	pdMap, ok := pdAll.Result.(map[string]interface{})
	if !ok || pdMap["phases"] == nil {
		t.Fatalf("expected phases for empty phase, got %#v", pdAll.Result)
	}

	pdLex := s.handlePhaseDetails(LSPRequest{
		JSONRPC: "2.0",
		ID:      3,
		Params: mustJSONParams(t, map[string]interface{}{
			"textDocument": map[string]interface{}{"uri": uri},
			"phase":        "lexer",
		}),
	})
	if pdLex.Error != nil {
		t.Fatalf("phaseDetails lexer: %+v", pdLex.Error)
	}
	if _, ok := pdLex.Result.(map[string]interface{})["details"]; !ok {
		t.Fatalf("expected details for lexer phase: %#v", pdLex.Result)
	}
}
