package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"forst/internal/discovery"
	"forst/internal/httpbody"
	"forst/internal/invokedispatch"
)

// jsonMarshalVersionPayload marshals the /version payload; tests may replace to inject errors.
var jsonMarshalVersionPayload = json.Marshal

// jsonMarshalFunctionsList encodes the function list for /functions; tests may replace to inject errors.
var jsonMarshalFunctionsList = json.Marshal

func (s *DevServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.invoke.HandleHealth(w, r)
}

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
		"runtime":         "dev",
	}
	data, err := jsonMarshalVersionPayload(payload)
	if err != nil {
		s.sendError(w, fmt.Sprintf("failed to marshal version: %v", err), http.StatusInternalServerError)
		return
	}
	s.sendJSONResponse(w, DevServerResponse{Success: true, Result: data})
}

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

	response := DevServerResponse{Success: true}
	if resultData, err := jsonMarshalFunctionsList(functions); err == nil {
		response.Result = resultData
	}
	s.sendJSONResponse(w, response)
}

func (s *DevServer) handleInvoke(w http.ResponseWriter, r *http.Request) {
	s.applyTestBackend(s.functions, s.fnExec)
	s.invoke.HandleInvoke(w, r)
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

	s.sendJSONResponse(w, DevServerResponse{Success: true, Output: typesContent})
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
	response := DevServerResponse{Success: false, Error: errorMsg}
	w.WriteHeader(statusCode)
	s.sendJSONResponse(w, response)
}

// applyTestBackend wires test doubles into the invoke server (dev_server_http_test).
func (s *DevServer) applyTestBackend(functions map[string]map[string]discovery.FunctionInfo, exec devFunctionExecutor) {
	stub := &testInvokeBackend{functions: functions, exec: exec}
	s.setInvokeBackendForTest(stub)
	s.mu.Lock()
	s.functions = functions
	s.mu.Unlock()
}

type testInvokeBackend struct {
	functions map[string]map[string]discovery.FunctionInfo
	exec      devFunctionExecutor
}

func (b *testInvokeBackend) Functions() map[string]map[string]discovery.FunctionInfo {
	return b.functions
}

func (b *testInvokeBackend) RefreshFunctions(context.Context) error {
	return nil
}

func (b *testInvokeBackend) Invoke(ctx context.Context, pkg, fn string, args json.RawMessage) (*invokedispatch.InvokeResult, error) {
	if b.exec == nil {
		return nil, fmt.Errorf("no executor")
	}
	result, err := b.exec.ExecuteFunction(pkg, fn, args)
	if err != nil {
		return nil, err
	}
	return &invokedispatch.InvokeResult{
		Success: result.Success,
		Output:  result.Output,
		Error:   result.Error,
		Result:  result.Result,
	}, nil
}

func (b *testInvokeBackend) InvokeStream(ctx context.Context, pkg, fn string, args json.RawMessage) (<-chan invokedispatch.StreamChunk, error) {
	if b.exec == nil {
		return nil, fmt.Errorf("no executor")
	}
	ch, err := b.exec.ExecuteStreamingFunction(ctx, pkg, fn, args)
	if err != nil {
		return nil, err
	}
	out := make(chan invokedispatch.StreamChunk)
	go func() {
		defer close(out)
		for item := range ch {
			out <- invokedispatch.StreamChunk{Data: item.Data, Status: item.Status, Error: item.Error}
		}
	}()
	return out, nil
}

var _ = httpbody.DefaultMaxBytes
