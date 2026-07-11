package nodert

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCallAsync_fulfilledReturnsValue(t *testing.T) {
	t.Helper()
	client, server := pairedCallAsyncClient(t, func(req Request) Response {
		if req.Method == MethodInitialize {
			return okResponse(req)
		}
		if req.Method != MethodCallAsync {
			return okResponse(req)
		}
		return Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Result:  json.RawMessage(`{"value":{"id":"pay_123","amount":42}}`),
		}
	})
	defer func() { _ = server.Close() }()

	type result struct {
		ID     string  `json:"id"`
		Amount float64 `json:"amount"`
	}
	got, err := callAsyncValue[result](client, "legacy/payment.ts", "create", 42, "usd")
	if err != nil {
		t.Fatalf("CallAsync: %v", err)
	}
	if got.ID != "pay_123" || got.Amount != 42 {
		t.Fatalf("result = %#v", got)
	}
}

func TestCallAsync_rejectionPreservesStack(t *testing.T) {
	t.Helper()
	client, server := pairedCallAsyncClient(t, func(req Request) Response {
		if req.Method == MethodInitialize {
			return okResponse(req)
		}
		if req.Method != MethodCallAsync {
			return okResponse(req)
		}
		return Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Error: &ResponseError{
				Code:    ErrCodeServerError,
				Message: "Payment provider timeout",
				Data: json.RawMessage(`{
					"name": "TimeoutError",
					"stack": "TimeoutError: Payment provider timeout\n    at failWithError (payment.ts:12:11)",
					"moduleId": "legacy/payment.ts",
					"exportName": "failWithError"
				}`),
			},
		}
	})
	defer func() { _ = server.Close() }()

	_, err := callAsyncValue[map[string]string](client, "legacy/payment.ts", "failWithError")
	if err == nil {
		t.Fatal("expected error")
	}
	var nodeErr *NodeCallError
	if !AsNodeCallError(err, &nodeErr) {
		t.Fatalf("expected NodeCallError, got %T: %v", err, err)
	}
	if nodeErr.Message != "Payment provider timeout" {
		t.Fatalf("message = %q", nodeErr.Message)
	}
	if !strings.Contains(nodeErr.Stack, "payment.ts:12:11") {
		t.Fatalf("stack missing source location: %q", nodeErr.Stack)
	}
	if nodeErr.ModuleID != "legacy/payment.ts" || nodeErr.ExportName != "failWithError" {
		t.Fatalf("metadata = %+v", nodeErr)
	}
	if !strings.Contains(err.Error(), "payment.ts:12:11") {
		t.Fatalf("Error() text missing stack: %q", err.Error())
	}
}

func TestCallAsync_concurrentGoroutines_noCrossWire(t *testing.T) {
	t.Helper()
	var mu sync.Mutex
	client, server := pairedCallAsyncClient(t, func(req Request) Response {
		if req.Method == MethodInitialize {
			return okResponse(req)
		}
		if req.Method != MethodCallAsync {
			return okResponse(req)
		}
		var params CallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			t.Errorf("unmarshal params: %v", err)
		}
		var args []json.RawMessage
		if err := json.Unmarshal(params.Args, &args); err != nil {
			t.Errorf("unmarshal args: %v", err)
		}
		var n int
		if len(args) > 0 {
			_ = json.Unmarshal(args[0], &n)
		}
		mu.Lock()
		defer mu.Unlock()
		return Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Result:  json.RawMessage(fmt.Sprintf(`{"value":{"echo":%d}}`, n)),
		}
	})
	defer func() { _ = server.Close() }()

	const nCalls = 16
	var wg sync.WaitGroup
	errs := make([]error, nCalls)
	results := make([]int, nCalls)
	for i := range nCalls {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			type echoResult struct {
				Echo int `json:"echo"`
			}
			got, err := callAsyncValue[echoResult](client, "legacy/payment.ts", "concurrentEcho", i)
			errs[i] = err
			if err == nil {
				results[i] = got.Echo
			}
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", i, err)
		}
		if results[i] != i {
			t.Fatalf("goroutine %d: echo = %d want %d", i, results[i], i)
		}
	}
}

func TestCallAsync_timeout_cleansPending(t *testing.T) {
	t.Helper()
	client, server := pairedCallAsyncClient(t, func(req Request) Response {
		if req.Method == MethodInitialize {
			return okResponse(req)
		}
		if req.Method == MethodCallAsync {
			time.Sleep(200 * time.Millisecond)
		}
		return okResponse(req)
	})
	defer func() { _ = server.Close() }()
	client.SetCallTimeout(25 * time.Millisecond)

	_, err := callAsyncValue[map[string]string](client, "legacy/payment.ts", "delayed")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") && !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected timeout, got %v", err)
	}
	if client.PendingCount() != 0 {
		t.Fatalf("pending map not cleaned: count = %d", client.PendingCount())
	}
	time.Sleep(250 * time.Millisecond)
}

func pairedCallAsyncClient(t *testing.T, handler func(Request) Response) (*Client, *mockServer) {
	t.Helper()
	client, server := pairedClientServer(t, handler)
	if err := client.Initialize(sampleManifestAsync(), nil); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	return client, server
}

func callAsyncValue[T any](client *Client, moduleID, exportName string, args ...any) (T, error) {
	var zero T
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return zero, err
	}
	raw, err := client.CallAsync(moduleID, exportName, argsJSON)
	if err != nil {
		return zero, err
	}
	var out T
	if len(raw) == 0 || string(raw) == "null" {
		return out, nil
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return zero, fmt.Errorf("unmarshal node call result: %w", err)
	}
	return out, nil
}

func sampleManifestAsync() Manifest {
	return Manifest{
		Version:      ManifestVersion,
		BoundaryRoot: "/tmp/project",
		Exports: []ExportEntry{
			{ModuleID: "legacy/payment.ts", Name: "create", Kind: ExportKindAsyncFunction},
			{ModuleID: "legacy/payment.ts", Name: "failWithError", Kind: ExportKindAsyncFunction},
			{ModuleID: "legacy/payment.ts", Name: "concurrentEcho", Kind: ExportKindAsyncFunction},
			{ModuleID: "legacy/payment.ts", Name: "delayed", Kind: ExportKindAsyncFunction},
		},
	}
}
