package main

import (
	"fmt"
	"net/http"
	"time"
)

// handleTypes returns shared shape types from the generated client (dist/types.d.ts).
func (s *DevServer) handleTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	forceRegenerate := r.URL.Query().Get("force") == "true"

	s.typesCacheMu.RLock()
	_, exists := s.typesCache["types"]
	lastGen := s.lastTypesGen
	s.typesCacheMu.RUnlock()

	shouldRegenerate := forceRegenerate || !exists || time.Since(lastGen) > 5*time.Minute

	if shouldRegenerate {
		s.log.Debug("Reading TypeScript types from generated client...")

		typesContent, err := s.readGeneratedTypesContent()
		if err != nil {
			s.sendError(w, fmt.Sprintf("Failed to read generated TypeScript types: %v", err), http.StatusInternalServerError)
			return
		}

		s.typesCacheMu.Lock()
		s.typesCache["types"] = typesContent
		s.lastTypesGen = time.Now()
		s.typesCacheMu.Unlock()

		s.log.Debug("TypeScript types loaded from .forst/client")
	} else {
		s.log.Debug("Using cached TypeScript types")
	}

	s.typesCacheMu.RLock()
	typesContent := s.typesCache["types"]
	s.typesCacheMu.RUnlock()

	s.sendJSONResponse(w, DevServerResponse{Success: true, Output: typesContent})
}
