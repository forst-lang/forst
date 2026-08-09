package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"forst/internal/discovery"
	"forst/internal/httpbody"
	"forst/internal/invokedispatch"
	"forst/internal/invokeserver"
)

func (s *DevServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.invoke.HandleHealth(w, r)
}

func (s *DevServer) handleVersion(w http.ResponseWriter, r *http.Request) {
	s.invoke.HandleVersion(w, r)
}

func (s *DevServer) handleFunctions(w http.ResponseWriter, r *http.Request) {
	s.invoke.HandleFunctions(w, r)
}

func (s *DevServer) handleInvoke(w http.ResponseWriter, r *http.Request) {
	s.applyTestBackend(s.functions, s.fnExec)
	s.invoke.HandleInvoke(w, r)
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
	functions  map[string]map[string]discovery.FunctionInfo
	exec       devFunctionExecutor
	refreshErr error
}

func (b *testInvokeBackend) Functions() map[string]map[string]discovery.FunctionInfo {
	return b.functions
}

func (b *testInvokeBackend) RefreshFunctions(context.Context) error {
	return b.refreshErr
}

func (b *testInvokeBackend) Invoke(ctx context.Context, pkg, fn string, args json.RawMessage) (*invokedispatch.InvokeResult, error) {
	if b.exec == nil {
		return nil, fmt.Errorf("no executor")
	}
	result, err := b.exec.ExecuteFunction(ctx, pkg, fn, args)
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
	return invokeserver.AdaptExecutorStream(ch), nil
}

var _ = httpbody.DefaultMaxBytes
