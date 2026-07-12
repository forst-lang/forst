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
	invokeFn func(ctx context.Context, pkg, fn string, args json.RawMessage) (*invokedispatch.InvokeResult, error)
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

func (b *stubInvokeBackend) InvokeStream(context.Context, string, string, json.RawMessage) (<-chan invokedispatch.StreamChunk, error) {
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
