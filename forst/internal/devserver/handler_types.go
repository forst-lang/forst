package devserver

import (
	"fmt"
	"net/http"
	"time"
)

// handleTypes handles TypeScript type generation requests.
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
		s.log.Debug("Regenerating TypeScript types...")

		if err := s.discoverFunctions(); err != nil {
			s.sendError(w, fmt.Sprintf("Failed to discover functions: %v", err), http.StatusInternalServerError)
			return
		}

		s.mu.RLock()
		functions := s.functions
		s.mu.RUnlock()

		typesContent, err := s.typesGenerator.GenerateTypesForFunctions(functions, s.discoverer.GetRootDir())
		if err != nil {
			s.sendError(w, fmt.Sprintf("Failed to generate TypeScript types: %v", err), http.StatusInternalServerError)
			return
		}

		s.typesCacheMu.Lock()
		s.typesCache["types"] = typesContent
		s.lastTypesGen = time.Now()
		s.typesCacheMu.Unlock()

		s.log.Debug("TypeScript types generated and cached")
	} else {
		s.log.Debug("Using cached TypeScript types")
	}

	s.typesCacheMu.RLock()
	typesContent := s.typesCache["types"]
	s.typesCacheMu.RUnlock()

	response := DevServerResponse{
		Success: true,
		Output:  typesContent,
	}

	s.sendJSONResponse(w, response)
}
