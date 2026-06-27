package devserver

import "net/http"

// handleHealth handles health check requests.
func (s *DevServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := DevServerResponse{
		Success: true,
		Output:  "Forst HTTP server is healthy",
	}

	s.sendJSONResponse(w, response)
}
