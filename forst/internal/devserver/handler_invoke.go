package devserver

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// handleInvoke handles function invocation requests.
func (s *DevServer) handleInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req InvokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, fmt.Sprintf("Failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	s.invokeWithArgs(w, r, req.Package, req.Function, req.Args, req.Streaming)
}
