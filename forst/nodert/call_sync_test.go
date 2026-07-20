package nodert

import (
	"encoding/json"
	"testing"

	logrus "github.com/sirupsen/logrus"
)

func setTestSupervisorClient(t *testing.T, client *Client) {
	t.Helper()
	resetSupervisorForTest()
	t.Cleanup(func() {
		supervisorMu.Lock()
		supervisorInst = nil
		supervisorMu.Unlock()
	})
	supervisorMu.Lock()
	supervisorInst = &Supervisor{
		client: client,
		log:    logrus.New(),
	}
	supervisorMu.Unlock()
}

func TestCallSync_nullResultReturnsZeroValue(t *testing.T) {
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

	got, err := CallSync[int]("legacy/payment.ts", "create")
	if err != nil {
		t.Fatalf("CallSync: %v", err)
	}
	if got != 0 {
		t.Fatalf("got = %d, want zero value", got)
	}
}

func TestCallSync_emptyResultReturnsZeroValue(t *testing.T) {
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
			return Response{JSONRPC: JSONRPCVersion, ID: req.ID, Result: json.RawMessage(`""`)}
		default:
			return okResponse(req)
		}
	})
	defer func() { _ = server.Close() }()
	if err := client.Initialize(sampleManifest(), nil); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	setTestSupervisorClient(t, client)

	got, err := CallSync[string]("legacy/payment.ts", "create")
	if err != nil {
		t.Fatalf("CallSync: %v", err)
	}
	if got != "" {
		t.Fatalf("got = %q, want empty string", got)
	}
}

func TestMustCallSync_panicsOnError(t *testing.T) {
	t.Parallel()
	resetSupervisorForTest()
	t.Cleanup(resetSupervisorForTest)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	MustCallSync[int]("legacy/payment.ts", "create")
}
