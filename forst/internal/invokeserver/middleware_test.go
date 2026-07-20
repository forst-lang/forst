package invokeserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"forst/internal/discovery"
	"forst/internal/invokedispatch"
	"forst/internal/reload"
)

type stubInvokeBackend struct {
	invokeFn       func(ctx context.Context, pkg, fn string, args json.RawMessage) (*invokedispatch.InvokeResult, error)
	invokeStreamFn func(ctx context.Context, pkg, fn string, args json.RawMessage) (<-chan invokedispatch.StreamChunk, error)
}

func (b *stubInvokeBackend) Functions() map[string]map[string]discovery.FunctionInfo {
	return nil
}

func (b *stubInvokeBackend) RefreshFunctions(context.Context) error {
	return nil
}

func (b *stubInvokeBackend) Invoke(ctx context.Context, pkg, fn string, args json.RawMessage) (*invokedispatch.InvokeResult, error) {
	if b.invokeFn != nil {
		return b.invokeFn(ctx, pkg, fn, args)
	}
	return &invokedispatch.InvokeResult{Success: true}, nil
}

func (b *stubInvokeBackend) InvokeStream(ctx context.Context, pkg, fn string, args json.RawMessage) (<-chan invokedispatch.StreamChunk, error) {
	if b.invokeStreamFn != nil {
		return b.invokeStreamFn(ctx, pkg, fn, args)
	}
	return nil, errors.New("not implemented")
}

func TestReloadGate_blocksInvokeWhenDraining(t *testing.T) {
	t.Parallel()
	coord := reload.NewCoordinator()
	coord.SetState(reload.StateDraining, nil)

	backend := WrapBackend(&stubInvokeBackend{}, WithReloadGate(coord))
	_, err := backend.Invoke(context.Background(), "pkg", "Fn", nil)
	if !errors.Is(err, ErrReloading) {
		t.Fatalf("Invoke() err = %v, want ErrReloading", err)
	}
}

func TestReloadGate_allowsInvokeWhenReady(t *testing.T) {
	t.Parallel()
	coord := reload.NewCoordinator()

	var called bool
	backend := WrapBackend(&stubInvokeBackend{
		invokeFn: func(context.Context, string, string, json.RawMessage) (*invokedispatch.InvokeResult, error) {
			called = true
			return &invokedispatch.InvokeResult{Success: true, Result: json.RawMessage(`{"ok":true}`)}, nil
		},
	}, WithReloadGate(coord))

	result, err := backend.Invoke(context.Background(), "pkg", "Fn", nil)
	if err != nil {
		t.Fatalf("Invoke() err = %v", err)
	}
	if !called {
		t.Fatal("backend Invoke was not called")
	}
	if !result.Success {
		t.Fatalf("result = %+v", result)
	}
}

func TestReloadGate_blocksInvokeStreamWhenNotReady(t *testing.T) {
	t.Parallel()
	coord := reload.NewCoordinator()
	coord.SetState(reload.StateRegenerating, nil)

	backend := WrapBackend(&stubInvokeBackend{}, WithReloadGate(coord))
	_, err := backend.InvokeStream(context.Background(), "pkg", "Fn", nil)
	if !errors.Is(err, ErrReloading) {
		t.Fatalf("InvokeStream() err = %v, want ErrReloading", err)
	}
}

func TestReloadGate_propagatesDegradedError(t *testing.T) {
	t.Parallel()
	coord := reload.NewCoordinator()
	degradedErr := errors.New("compile failed")
	coord.SetState(reload.StateDegraded, degradedErr)

	backend := WrapBackend(&stubInvokeBackend{}, WithReloadGate(coord))
	_, err := backend.Invoke(context.Background(), "pkg", "Fn", nil)
	if errors.Is(err, ErrReloading) {
		t.Fatalf("Invoke() err = %v, want degraded error not ErrReloading", err)
	}
	if !errors.Is(err, degradedErr) {
		t.Fatalf("Invoke() err = %v, want %v", err, degradedErr)
	}
}

func TestReloadGate_propagatesDegradedError_onStream(t *testing.T) {
	t.Parallel()
	coord := reload.NewCoordinator()
	degradedErr := errors.New("compile failed")
	coord.SetState(reload.StateDegraded, degradedErr)

	backend := WrapBackend(&stubInvokeBackend{}, WithReloadGate(coord))
	_, err := backend.InvokeStream(context.Background(), "pkg", "Fn", nil)
	if errors.Is(err, ErrReloading) {
		t.Fatalf("InvokeStream() err = %v, want degraded error not ErrReloading", err)
	}
	if !errors.Is(err, degradedErr) {
		t.Fatalf("InvokeStream() err = %v, want %v", err, degradedErr)
	}
}
