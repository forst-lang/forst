package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"forst/internal/discovery"
)

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

// handleVersion returns compiler build metadata and the dev HTTP contract version.
func (s *DevServer) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	payload := map[string]string{
		"version":         Version,
		"commit":          Commit,
		"date":            Date,
		"contractVersion": devHTTPContractVersion,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		s.sendError(w, fmt.Sprintf("failed to marshal version: %v", err), http.StatusInternalServerError)
		return
	}
	response := DevServerResponse{
		Success: true,
		Result:  data,
	}
	s.sendJSONResponse(w, response)
}

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

	if resultData, err := json.Marshal(functions); err == nil {
		response.Result = resultData
	}

	s.sendJSONResponse(w, response)
}

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

	s.mu.RLock()
	pkgFuncs, ok := s.functions[req.Package]
	if !ok {
		s.mu.RUnlock()
		s.sendError(w, fmt.Sprintf("Package %s not found", req.Package), http.StatusNotFound)
		return
	}

	fn, ok := pkgFuncs[req.Function]
	s.mu.RUnlock()
	if !ok {
		s.sendError(w, fmt.Sprintf("Function %s not found in package %s", req.Function, req.Package), http.StatusNotFound)
		return
	}

	if req.Streaming && !fn.SupportsStreaming {
		s.sendError(w, fmt.Sprintf("Function %s does not support streaming", req.Function), http.StatusBadRequest)
		return
	}

	if req.Streaming {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Transfer-Encoding", "chunked")
		flusher, ok := w.(http.Flusher)
		if !ok {
			s.sendError(w, "Streaming not supported by server", http.StatusInternalServerError)
			return
		}

		results, err := s.fnExec.ExecuteStreamingFunction(r.Context(), req.Package, req.Function, req.Args)
		if err != nil {
			s.sendError(w, fmt.Sprintf("Streaming execution failed: %v", err), http.StatusInternalServerError)
			return
		}

		encoder := json.NewEncoder(w)
		for result := range results {
			if err := encoder.Encode(result); err != nil {
				s.log.Errorf("Failed to encode streaming result: %v", err)
				return
			}
			flusher.Flush()
		}
	} else {
		result, err := s.fnExec.ExecuteFunction(req.Package, req.Function, req.Args)
		if err != nil {
			s.sendError(w, fmt.Sprintf("Function execution failed: %v", err), http.StatusInternalServerError)
			return
		}

		response := DevServerResponse{
			Success: result.Success,
			Output:  result.Output,
			Error:   result.Error,
			Result:  result.Result,
		}
		s.sendJSONResponse(w, response)
	}
}

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

// sendJSONResponse sends a JSON response to the client.
func (s *DevServer) sendJSONResponse(w http.ResponseWriter, response DevServerResponse) {
	w.Header().Set("Content-Type", "application/json")

	if s.config.Server.CORS {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	}

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

// sendError sends an error response to the client.
func (s *DevServer) sendError(w http.ResponseWriter, errorMsg string, statusCode int) {
	response := DevServerResponse{
		Success: false,
		Error:   errorMsg,
	}

	w.WriteHeader(statusCode)
	s.sendJSONResponse(w, response)
}
