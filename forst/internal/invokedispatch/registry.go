package invokedispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"forst/internal/discovery"
)

// Handler invokes a Forst function from decoded JSON args.
type Handler func(args json.RawMessage) (any, error)

// InvokeResult mirrors executor.ExecutionResult for HTTP envelope mapping.
type InvokeResult struct {
	Success bool            `json:"success"`
	Output  string          `json:"output,omitempty"`
	Error   string          `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

// StreamChunk is one NDJSON line for streaming invoke responses.
type StreamChunk struct {
	Data   any    `json:"data"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// Registry maps (package, function) to handlers and metadata.
type Registry struct {
	mu        sync.RWMutex
	functions map[string]map[string]discovery.FunctionInfo
	handlers  map[string]Handler
}

// NewRegistry creates an empty invoke registry.
func NewRegistry() *Registry {
	return &Registry{
		functions: make(map[string]map[string]discovery.FunctionInfo),
		handlers:  make(map[string]Handler),
	}
}

func registryKey(pkg, fn string) string {
	return pkg + "." + fn
}

// FunctionMeta is compile-time metadata for an invokable Forst export (embedded invoke companion).
type FunctionMeta struct {
	Package           string
	Name              string
	SupportsStreaming bool
	Runnable          bool
}

// RegisterMeta adds a callable function and its metadata.
func (r *Registry) RegisterMeta(fn FunctionMeta, h Handler) {
	r.Register(discovery.FunctionInfo{
		Package:           fn.Package,
		Name:              fn.Name,
		SupportsStreaming: fn.SupportsStreaming,
		Runnable:          fn.Runnable,
	}, h)
}

// Register adds a callable function and its metadata.
func (r *Registry) Register(fn discovery.FunctionInfo, h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.functions[fn.Package] == nil {
		r.functions[fn.Package] = make(map[string]discovery.FunctionInfo)
	}
	r.functions[fn.Package][fn.Name] = fn
	r.handlers[registryKey(fn.Package, fn.Name)] = h
}

// Functions returns a snapshot of registered function metadata.
func (r *Registry) Functions() map[string]map[string]discovery.FunctionInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]map[string]discovery.FunctionInfo, len(r.functions))
	for pkg, fns := range r.functions {
		out[pkg] = make(map[string]discovery.FunctionInfo, len(fns))
		for name, info := range fns {
			out[pkg][name] = info
		}
	}
	return out
}

// Invoke calls a registered handler and JSON-encodes the result.
func (r *Registry) Invoke(_ context.Context, pkg, fn string, args json.RawMessage) (*InvokeResult, error) {
	r.mu.RLock()
	meta, ok := r.functions[pkg][fn]
	h := r.handlers[registryKey(pkg, fn)]
	r.mu.RUnlock()
	if !ok || h == nil {
		return nil, fmt.Errorf("function %s.%s not found", pkg, fn)
	}
	if args == nil {
		args = json.RawMessage("[]")
	}
	value, err := h(args)
	if err != nil {
		return &InvokeResult{Success: false, Error: err.Error()}, nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	_ = meta
	return &InvokeResult{Success: true, Result: raw}, nil
}

// InvokeStream is not supported for embedded in-process handlers in phase 1.
func (r *Registry) InvokeStream(_ context.Context, pkg, fn string, _ json.RawMessage) (<-chan StreamChunk, error) {
	r.mu.RLock()
	meta, ok := r.functions[pkg][fn]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("function %s.%s not found", pkg, fn)
	}
	if meta.SupportsStreaming {
		return nil, fmt.Errorf("streaming invoke is not supported for embedded invoke yet")
	}
	return nil, fmt.Errorf("function %s.%s does not support streaming", pkg, fn)
}
