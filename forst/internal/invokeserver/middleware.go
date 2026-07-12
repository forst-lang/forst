package invokeserver

import (
	"context"
	"encoding/json"
	"errors"

	"forst/internal/discovery"
	"forst/internal/invokedispatch"
	"forst/internal/reload"
)

// ErrReloading is returned when invoke is blocked during reload drain/regeneration.
var ErrReloading = errors.New("server reloading")

// BackendMiddleware wraps a DispatchBackend.
type BackendMiddleware func(DispatchBackend) DispatchBackend

// WrapBackend applies middlewares outer-to-inner around backend.
func WrapBackend(backend DispatchBackend, middlewares ...BackendMiddleware) DispatchBackend {
	for i := len(middlewares) - 1; i >= 0; i-- {
		backend = middlewares[i](backend)
	}
	return backend
}

// WithReloadGate blocks invoke while the coordinator is draining or regenerating
// and tracks in-flight invoke work for drain.
func WithReloadGate(coord *reload.ReloadCoordinator) BackendMiddleware {
	return func(next DispatchBackend) DispatchBackend {
		return &reloadGateBackend{coord: coord, backend: next}
	}
}

type reloadGateBackend struct {
	coord   *reload.ReloadCoordinator
	backend DispatchBackend
}

func (b *reloadGateBackend) Functions() map[string]map[string]discovery.FunctionInfo {
	return b.backend.Functions()
}

func (b *reloadGateBackend) RefreshFunctions(ctx context.Context) error {
	return b.backend.RefreshFunctions(ctx)
}

func (b *reloadGateBackend) Invoke(ctx context.Context, pkg, fn string, args json.RawMessage) (*invokedispatch.InvokeResult, error) {
	if err := b.coord.WaitReady(ctx); err != nil {
		if errors.Is(err, reload.ErrNotReady) {
			return nil, ErrReloading
		}
		return nil, err
	}
	var (
		result *invokedispatch.InvokeResult
		err    error
	)
	b.coord.TrackInFlight(func() {
		result, err = b.backend.Invoke(ctx, pkg, fn, args)
	})
	return result, err
}

func (b *reloadGateBackend) InvokeStream(ctx context.Context, pkg, fn string, args json.RawMessage) (<-chan invokedispatch.StreamChunk, error) {
	if err := b.coord.WaitReady(ctx); err != nil {
		if errors.Is(err, reload.ErrNotReady) {
			return nil, ErrReloading
		}
		return nil, err
	}
	var (
		ch  <-chan invokedispatch.StreamChunk
		err error
	)
	b.coord.TrackInFlight(func() {
		ch, err = b.backend.InvokeStream(ctx, pkg, fn, args)
	})
	return ch, err
}
