package devserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// handleInvokeRaw is like handleInvoke but the POST body is **only** the JSON array of arguments.
// Package and function come from query parameters: package, function, optional streaming=true.
// Node clients can forward the raw request body (e.g. after express.raw) without parsing into V8 objects first.
func (s *DevServer) handleInvokeRaw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pkg := r.URL.Query().Get("package")
	fn := r.URL.Query().Get("function")
	if pkg == "" || fn == "" {
		s.sendError(w, "query parameters package and function are required", http.StatusBadRequest)
		return
	}
	streaming := r.URL.Query().Get("streaming") == "true"

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		s.sendError(w, fmt.Sprintf("read body: %v", err), http.StatusBadRequest)
		return
	}
	if len(raw) == 0 {
		s.sendError(w, "body is required (JSON array of arguments)", http.StatusBadRequest)
		return
	}
	if !json.Valid(raw) {
		s.sendError(w, "body must be valid JSON", http.StatusBadRequest)
		return
	}
	var probe []json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		s.sendError(w, "body must be a JSON array", http.StatusBadRequest)
		return
	}

	s.invokeWithArgs(w, r, pkg, fn, json.RawMessage(raw), streaming)
}
