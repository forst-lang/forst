package invokeserver

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"forst/internal/discovery"
	"forst/internal/executor"
	"forst/internal/invokedispatch"
)

// DispatchBackend executes invoke requests.
type DispatchBackend interface {
	Functions() map[string]map[string]discovery.FunctionInfo
	RefreshFunctions(ctx context.Context) error
	Invoke(ctx context.Context, pkg, fn string, args json.RawMessage) (*invokedispatch.InvokeResult, error)
	InvokeStream(ctx context.Context, pkg, fn string, args json.RawMessage) (<-chan invokedispatch.StreamChunk, error)
}

type functionDiscoverer interface {
	DiscoverFunctions() (map[string]map[string]discovery.FunctionInfo, error)
}

type devFunctionExecutor interface {
	ExecuteFunction(ctx context.Context, packageName, functionName string, args json.RawMessage) (*executor.ExecutionResult, error)
	ExecuteStreamingFunction(ctx context.Context, packageName, functionName string, args json.RawMessage) (<-chan executor.StreamingResult, error)
}

// DevBackend wraps the dev FunctionExecutor and discovery.
type DevBackend struct {
	exec       devFunctionExecutor
	discoverer functionDiscoverer
	functions  map[string]map[string]discovery.FunctionInfo
	mu         sync.RWMutex
}

// NewDevBackend creates a backend that recompiles and go-runs on each invoke.
func NewDevBackend(exec *executor.FunctionExecutor, discoverer *discovery.Discoverer) *DevBackend {
	return &DevBackend{
		exec:       exec,
		discoverer: discoverer,
		functions:  make(map[string]map[string]discovery.FunctionInfo),
	}
}

// RefreshFunctions re-discovers public functions.
func (b *DevBackend) RefreshFunctions(_ context.Context) error {
	functions, err := b.discoverer.DiscoverFunctions()
	if err != nil {
		return fmt.Errorf("discover functions: %w", err)
	}
	b.mu.Lock()
	b.functions = functions
	b.mu.Unlock()
	return nil
}

// Functions returns the cached function map.
func (b *DevBackend) Functions() map[string]map[string]discovery.FunctionInfo {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make(map[string]map[string]discovery.FunctionInfo, len(b.functions))
	for pkg, fns := range b.functions {
		out[pkg] = make(map[string]discovery.FunctionInfo, len(fns))
		for name, info := range fns {
			out[pkg][name] = info
		}
	}
	return out
}

// Invoke runs a function via the dev executor.
func (b *DevBackend) Invoke(ctx context.Context, pkg, fn string, args json.RawMessage) (*invokedispatch.InvokeResult, error) {
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

// InvokeStream runs a streaming function via the dev executor.
func (b *DevBackend) InvokeStream(ctx context.Context, pkg, fn string, args json.RawMessage) (<-chan invokedispatch.StreamChunk, error) {
	ch, err := b.exec.ExecuteStreamingFunction(ctx, pkg, fn, args)
	if err != nil {
		return nil, err
	}
	return AdaptExecutorStream(ch), nil
}

// AdaptExecutorStream maps executor streaming results to invoke dispatch chunks.
func AdaptExecutorStream(ch <-chan executor.StreamingResult) <-chan invokedispatch.StreamChunk {
	out := make(chan invokedispatch.StreamChunk)
	go func() {
		defer close(out)
		for item := range ch {
			out <- invokedispatch.StreamChunk{
				Data:   item.Data,
				Status: item.Status,
				Error:  item.Error,
			}
		}
	}()
	return out
}

// RegistryBackend adapts invokedispatch.Registry to DispatchBackend.
type RegistryBackend struct {
	registry *invokedispatch.Registry
}

// NewRegistryBackend wraps a registry.
func NewRegistryBackend(registry *invokedispatch.Registry) *RegistryBackend {
	return &RegistryBackend{registry: registry}
}

// RefreshFunctions is a no-op for embedded registry.
func (b *RegistryBackend) RefreshFunctions(context.Context) error {
	return nil
}

// Functions returns registered metadata.
func (b *RegistryBackend) Functions() map[string]map[string]discovery.FunctionInfo {
	return b.registry.Functions()
}

// Invoke calls an in-process handler.
func (b *RegistryBackend) Invoke(ctx context.Context, pkg, fn string, args json.RawMessage) (*invokedispatch.InvokeResult, error) {
	return b.registry.Invoke(ctx, pkg, fn, args)
}

// InvokeStream delegates to the registry.
func (b *RegistryBackend) InvokeStream(ctx context.Context, pkg, fn string, args json.RawMessage) (<-chan invokedispatch.StreamChunk, error) {
	return b.registry.InvokeStream(ctx, pkg, fn, args)
}
