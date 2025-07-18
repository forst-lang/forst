package lsp

import (
	"encoding/json"
	"net/http"
	"time"
)

// handleHealth handles health check requests
func (s *LSPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "forst-lsp",
		"version":   Version,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}
