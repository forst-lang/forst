package devserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"forst/gateway"
)

// invokeWithArgs performs lookup and execution for POST /invoke and POST /invoke/raw.
func (s *DevServer) invokeWithArgs(w http.ResponseWriter, r *http.Request, pkg, fn string, args json.RawMessage, streaming bool) {
	s.mu.RLock()
	pkgFuncs, ok := s.functions[pkg]
	if !ok {
		s.mu.RUnlock()
		s.sendError(w, fmt.Sprintf("Package %s not found", pkg), http.StatusNotFound)
		return
	}

	finfo, ok := pkgFuncs[fn]
	s.mu.RUnlock()
	if !ok {
		s.sendError(w, fmt.Sprintf("Function %s not found in package %s", fn, pkg), http.StatusNotFound)
		return
	}

	if streaming && !finfo.SupportsStreaming {
		s.sendError(w, fmt.Sprintf("Function %s does not support streaming", fn), http.StatusBadRequest)
		return
	}

	if streaming {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Transfer-Encoding", "chunked")
		flusher, ok := w.(http.Flusher)
		if !ok {
			s.sendError(w, "Streaming not supported by server", http.StatusInternalServerError)
			return
		}

		results, err := s.fnExec.ExecuteStreamingFunction(r.Context(), pkg, fn, args)
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
		result, err := s.fnExec.ExecuteFunction(pkg, fn, args)
		if err != nil {
			s.sendError(w, fmt.Sprintf("Function execution failed: %v", err), http.StatusInternalServerError)
			return
		}

		if result.Success && finfo.IsGateway && len(result.Result) > 0 {
			if err := gateway.ValidateGatewayResultJSON(result.Result); err != nil {
				s.sendError(w, fmt.Sprintf("invalid gateway result: %v", err), http.StatusInternalServerError)
				return
			}
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
