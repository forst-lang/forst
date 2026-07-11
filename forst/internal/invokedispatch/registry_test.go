package invokedispatch

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"forst/internal/discovery"
)

func TestRegistry_registerAndFunctionsSnapshot(t *testing.T) {
	t.Helper()
	reg := NewRegistry()
	reg.RegisterMeta(FunctionMeta{
		Package:           "main",
		Name:              "Echo",
		SupportsStreaming: true,
		Runnable:          true,
	}, func(json.RawMessage) (any, error) {
		return "ok", nil
	})

	snapshot := reg.Functions()
	if got := snapshot["main"]["Echo"]; got.Name != "Echo" || !got.SupportsStreaming || !got.Runnable {
		t.Fatalf("unexpected metadata: %+v", got)
	}

	delete(snapshot["main"], "Echo")
	if _, ok := reg.Functions()["main"]["Echo"]; !ok {
		t.Fatal("Functions snapshot must not alias internal registry state")
	}
}

func TestRegistry_invokeSuccessAndNilArgs(t *testing.T) {
	t.Helper()
	reg := NewRegistry()
	reg.Register(discovery.FunctionInfo{Package: "demo", Name: "Ping", Runnable: true},
		func(json.RawMessage) (any, error) {
			return "pong", nil
		})
	reg.Register(discovery.FunctionInfo{Package: "demo", Name: "Add", Runnable: true},
		func(args json.RawMessage) (any, error) {
			var values []float64
			if err := json.Unmarshal(args, &values); err != nil {
				return nil, err
			}
			return values[0] + values[1], nil
		})

	result, err := reg.Invoke(context.Background(), "demo", "Ping", nil)
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if !result.Success || string(result.Result) != `"pong"` {
		t.Fatalf("unexpected nil-args result: %+v", result)
	}

	got, err := reg.Invoke(context.Background(), "demo", "Add", json.RawMessage(`[1,2]`))
	if err != nil {
		t.Fatalf("Invoke with args: %v", err)
	}
	if !got.Success || string(got.Result) != "3" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestRegistry_invokeHandlerError(t *testing.T) {
	t.Helper()
	reg := NewRegistry()
	reg.Register(discovery.FunctionInfo{Package: "demo", Name: "Fail", Runnable: true},
		func(json.RawMessage) (any, error) {
			return nil, errors.New("boom")
		})

	result, err := reg.Invoke(context.Background(), "demo", "Fail", json.RawMessage(`[]`))
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if result.Success || result.Error != "boom" {
		t.Fatalf("expected handler error envelope, got %+v", result)
	}
}

func TestRegistry_invokeNotFound(t *testing.T) {
	t.Helper()
	reg := NewRegistry()
	_, err := reg.Invoke(context.Background(), "missing", "Fn", json.RawMessage(`[]`))
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestRegistry_invokeMarshalError(t *testing.T) {
	t.Helper()
	reg := NewRegistry()
	reg.Register(discovery.FunctionInfo{Package: "demo", Name: "Bad", Runnable: true},
		func(json.RawMessage) (any, error) {
			return make(chan int), nil
		})

	_, err := reg.Invoke(context.Background(), "demo", "Bad", json.RawMessage(`[]`))
	if err == nil || !strings.Contains(err.Error(), "marshal result") {
		t.Fatalf("expected marshal error, got %v", err)
	}
}

func TestRegistry_invokeStream(t *testing.T) {
	t.Helper()

	tests := []struct {
		name    string
		meta    discovery.FunctionInfo
		wantErr string
	}{
		{
			name:    "not found",
			meta:    discovery.FunctionInfo{},
			wantErr: "not found",
		},
		{
			name: "streaming not supported yet",
			meta: discovery.FunctionInfo{
				Package:           "demo",
				Name:              "Stream",
				SupportsStreaming: true,
				Runnable:          true,
			},
			wantErr: "streaming invoke is not supported for embedded invoke yet",
		},
		{
			name: "no streaming support",
			meta: discovery.FunctionInfo{
				Package:  "demo",
				Name:     "Once",
				Runnable: true,
			},
			wantErr: "does not support streaming",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reg := NewRegistry()
			if tc.meta.Name != "" {
				reg.Register(tc.meta, func(json.RawMessage) (any, error) { return nil, nil })
			}
			_, err := reg.InvokeStream(context.Background(), tc.meta.Package, tc.meta.Name, json.RawMessage(`[]`))
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}
