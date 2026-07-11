package invokeserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"forst/internal/discovery"
	"forst/internal/executor"
	"forst/internal/invokedispatch"
)

type stubDiscoverer struct {
	functions map[string]map[string]discovery.FunctionInfo
	err       error
}

func (d *stubDiscoverer) DiscoverFunctions() (map[string]map[string]discovery.FunctionInfo, error) {
	if d.err != nil {
		return nil, d.err
	}
	return d.functions, nil
}

type stubDevExecutor struct {
	executeFn          func(packageName, functionName string, args json.RawMessage) (*executor.ExecutionResult, error)
	executeStreamingFn func(ctx context.Context, packageName, functionName string, args json.RawMessage) (<-chan executor.StreamingResult, error)
}

func (e *stubDevExecutor) ExecuteFunction(packageName, functionName string, args json.RawMessage) (*executor.ExecutionResult, error) {
	if e.executeFn != nil {
		return e.executeFn(packageName, functionName, args)
	}
	return nil, errors.New("no executeFn")
}

func (e *stubDevExecutor) ExecuteStreamingFunction(ctx context.Context, packageName, functionName string, args json.RawMessage) (<-chan executor.StreamingResult, error) {
	if e.executeStreamingFn != nil {
		return e.executeStreamingFn(ctx, packageName, functionName, args)
	}
	return nil, errors.New("no executeStreamingFn")
}

func newDevBackendWithStubs(d *stubDiscoverer, e *stubDevExecutor) *DevBackend {
	return &DevBackend{
		exec:       e,
		discoverer: d,
		functions:  make(map[string]map[string]discovery.FunctionInfo),
	}
}

func TestDevBackend_refreshFunctions_success(t *testing.T) {
	want := map[string]map[string]discovery.FunctionInfo{
		"main": {"Echo": {Package: "main", Name: "Echo"}},
	}
	b := newDevBackendWithStubs(&stubDiscoverer{functions: want}, &stubDevExecutor{})
	if err := b.RefreshFunctions(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := b.Functions()
	if got["main"]["Echo"].Name != "Echo" {
		t.Fatalf("Functions = %+v", got)
	}
}

func TestDevBackend_refreshFunctions_error(t *testing.T) {
	b := newDevBackendWithStubs(&stubDiscoverer{err: fmt.Errorf("discover")}, &stubDevExecutor{})
	err := b.RefreshFunctions(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDevBackend_functionsSnapshot(t *testing.T) {
	want := map[string]map[string]discovery.FunctionInfo{
		"main": {"Echo": {Package: "main", Name: "Echo"}},
	}
	b := newDevBackendWithStubs(&stubDiscoverer{functions: want}, &stubDevExecutor{})
	if err := b.RefreshFunctions(context.Background()); err != nil {
		t.Fatal(err)
	}
	snapshot := b.Functions()
	delete(snapshot["main"], "Echo")
	got := b.Functions()
	if _, ok := got["main"]["Echo"]; !ok {
		t.Fatal("snapshot must not alias internal state")
	}
}

func TestDevBackend_invoke_success(t *testing.T) {
	b := newDevBackendWithStubs(&stubDiscoverer{}, &stubDevExecutor{
		executeFn: func(_, _ string, _ json.RawMessage) (*executor.ExecutionResult, error) {
			return &executor.ExecutionResult{Success: true, Result: json.RawMessage(`{"ok":true}`)}, nil
		},
	})
	result, err := b.Invoke(context.Background(), "main", "Fn", json.RawMessage(`[]`))
	if err != nil || !result.Success || string(result.Result) != `{"ok":true}` {
		t.Fatalf("invoke: %+v err=%v", result, err)
	}
}

func TestDevBackend_invoke_error(t *testing.T) {
	b := newDevBackendWithStubs(&stubDiscoverer{}, &stubDevExecutor{
		executeFn: func(_, _ string, _ json.RawMessage) (*executor.ExecutionResult, error) {
			return nil, fmt.Errorf("exec failed")
		},
	})
	_, err := b.Invoke(context.Background(), "main", "Fn", json.RawMessage(`[]`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDevBackend_invokeStream_success(t *testing.T) {
	ch := make(chan executor.StreamingResult, 1)
	ch <- executor.StreamingResult{Data: "chunk", Status: "ok"}
	close(ch)
	b := newDevBackendWithStubs(&stubDiscoverer{}, &stubDevExecutor{
		executeStreamingFn: func(context.Context, string, string, json.RawMessage) (<-chan executor.StreamingResult, error) {
			return ch, nil
		},
	})
	out, err := b.InvokeStream(context.Background(), "main", "Stream", json.RawMessage(`[]`))
	if err != nil {
		t.Fatal(err)
	}
	chunk, ok := <-out
	if !ok || chunk.Status != "ok" || chunk.Data != "chunk" {
		t.Fatalf("chunk = %+v ok=%v", chunk, ok)
	}
	if _, more := <-out; more {
		t.Fatal("expected closed channel")
	}
}

func TestAdaptExecutorStream_forwardsChunks(t *testing.T) {
	ch := make(chan executor.StreamingResult, 2)
	ch <- executor.StreamingResult{Data: 1, Status: "a"}
	ch <- executor.StreamingResult{Data: 2, Status: "b", Error: "e"}
	close(ch)
	out := AdaptExecutorStream(ch)
	var chunks []invokedispatch.StreamChunk
	for c := range out {
		chunks = append(chunks, c)
	}
	if len(chunks) != 2 || chunks[1].Error != "e" {
		t.Fatalf("chunks = %+v", chunks)
	}
}

func TestRegistryBackend_invokeStream_delegates(t *testing.T) {
	reg := invokedispatch.NewRegistry()
	reg.Register(discovery.FunctionInfo{
		Package:           "demo",
		Name:              "Stream",
		SupportsStreaming: true,
		Runnable:          true,
	}, func(json.RawMessage) (any, error) { return nil, nil })
	backend := NewRegistryBackend(reg)
	_, err := backend.InvokeStream(context.Background(), "demo", "Stream", json.RawMessage(`[]`))
	if err == nil {
		t.Fatal("expected streaming not supported error from registry")
	}
}
