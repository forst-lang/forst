package nodert

import (
	"encoding/json"
	"testing"
)

func TestMarshalNodeCallArgsJSON_encodesPositionalArray(t *testing.T) {
	t.Parallel()
	raw, err := MarshalNodeCallArgsJSON(42, "usd")
	if err != nil {
		t.Fatalf("MarshalNodeCallArgsJSON: %v", err)
	}
	var got []any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0] != float64(42) {
		t.Fatalf("args[0] = %v", got[0])
	}
	if got[1] != "usd" {
		t.Fatalf("args[1] = %v", got[1])
	}
}

func TestCallSyncArgs_nullResultReturnsZeroValue(t *testing.T) {
	t.Parallel()
	client, server := pairedClientServer(t, func(req Request) Response {
		switch req.Method {
		case MethodInitialize:
			return Response{
				JSONRPC: JSONRPCVersion,
				ID:      req.ID,
				Result:  json.RawMessage(`{"ok":true,"protocol":"` + WireProtocolProtoV1 + `"}`),
			}
		case MethodCall:
			return Response{JSONRPC: JSONRPCVersion, ID: req.ID, Result: json.RawMessage(`null`)}
		default:
			return okResponse(req)
		}
	})
	defer func() { _ = server.Close() }()
	if err := client.Initialize(sampleManifest(), nil); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	setTestSupervisorClient(t, client)

	got, err := CallSyncArgs[string]("legacy/payment.ts", "create", json.RawMessage(`[1]`))
	if err != nil {
		t.Fatalf("CallSyncArgs: %v", err)
	}
	if got != "" {
		t.Fatalf("got = %q, want zero value", got)
	}
}
