package devserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"forst/internal/discovery"
)

// jsonMarshalFunctionsList encodes the function list for /functions; tests may replace to inject errors.
var jsonMarshalFunctionsList = json.Marshal

// handleFunctions handles function discovery requests.
func (s *DevServer) handleFunctions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.discoverFunctions(); err != nil {
		s.sendError(w, fmt.Sprintf("Failed to discover functions: %v", err), http.StatusInternalServerError)
		return
	}

	s.mu.RLock()
	functions := make([]discovery.FunctionInfo, 0)
	for _, pkgFuncs := range s.functions {
		for _, fn := range pkgFuncs {
			functions = append(functions, fn)
		}
	}
	s.mu.RUnlock()

	response := DevServerResponse{
		Success: true,
	}

	if resultData, err := jsonMarshalFunctionsList(functions); err == nil {
		response.Result = resultData
	}

	s.sendJSONResponse(w, response)
}
