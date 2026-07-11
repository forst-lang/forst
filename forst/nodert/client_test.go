package nodert

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"forst/nodert/pb"
)

func TestMain(m *testing.M) {
	code := m.Run()
	resetSupervisorForTest()
	os.Exit(code)
}

func sampleManifest() Manifest {
	return Manifest{
		Version:      ManifestVersion,
		BoundaryRoot: "/tmp/project",
		Exports: []ExportEntry{
			{ModuleID: "legacy/payment.ts", Name: "create", Kind: ExportKindFunction},
		},
	}
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("testdata", "rpc", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return strings.TrimSpace(string(data))
}

func okResponse(req Request) Response {
	if req.Method == MethodInitialize {
		return Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Result:  json.RawMessage(`{"ok":true,"protocol":"` + WireProtocolProtoV1 + `"}`),
		}
	}
	return Response{JSONRPC: JSONRPCVersion, ID: req.ID, Result: json.RawMessage(`{"ok":true}`)}
}

func TestClient_initializePingCallShutdown_happyPath(t *testing.T) {
	t.Helper()
	client, server := pairedClientServer(t, func(req Request) Response {
		switch req.Method {
		case MethodInitialize:
			return Response{JSONRPC: JSONRPCVersion, ID: req.ID, Result: json.RawMessage(`{"ok":true,"protocol":"` + WireProtocolProtoV1 + `"}`)}
		case MethodPing:
			return Response{JSONRPC: JSONRPCVersion, ID: req.ID, Result: json.RawMessage(`{"ok":true}`)}
		case MethodCall:
			return Response{JSONRPC: JSONRPCVersion, ID: req.ID, Result: json.RawMessage(`{"value":{"id":"pay_123"}}`)}
		case MethodShutdown:
			return Response{JSONRPC: JSONRPCVersion, ID: req.ID, Result: json.RawMessage(`{"ok":true}`)}
		default:
			return Response{
				JSONRPC: JSONRPCVersion,
				ID:      req.ID,
				Error:   &ResponseError{Code: ErrCodeMethodNotFound, Message: "Method not found"},
			}
		}
	})
	defer func() { _ = client.Shutdown() }()

	manifest := sampleManifest()
	if err := client.Initialize(manifest, nil); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if !client.Initialized() {
		t.Fatal("expected client initialized")
	}
	if err := client.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	value, err := client.CallSync("legacy/payment.ts", "create", json.RawMessage(`[42,"usd"]`))
	if err != nil {
		t.Fatalf("CallSync: %v", err)
	}
	var got map[string]string
	if err := json.Unmarshal(value, &got); err != nil {
		t.Fatalf("unmarshal value: %v", err)
	}
	if got["id"] != "pay_123" {
		t.Fatalf("call value = %#v", got)
	}
	if err := server.Close(); err != nil {
		t.Fatalf("close server: %v", err)
	}
}

func TestClient_callBeforeInitialize_rejected(t *testing.T) {
	t.Helper()
	client, server := pairedClientServer(t, func(req Request) Response {
		return okResponse(req)
	})
	defer func() { _ = server.Close() }()
	err := client.Ping()
	if err == nil || !strings.Contains(err.Error(), ErrInitializeRequired.Error()) {
		t.Fatalf("Ping before initialize: got %v want %v", err, ErrInitializeRequired)
	}
}

func TestClient_callAsync_rejectedWhenNotInitialized(t *testing.T) {
	t.Helper()
	client, server := pairedClientServer(t, func(req Request) Response {
		return okResponse(req)
	})
	defer func() { _ = server.Close() }()
	_, err := client.Call(MethodCallAsync, struct{}{})
	if err == nil || !strings.Contains(err.Error(), ErrInitializeRequired.Error()) {
		t.Fatalf("CallAsync before initialize: got %v want %v", err, ErrInitializeRequired)
	}
}

func TestClient_rpcForbiddenError(t *testing.T) {
	t.Helper()
	client, server := pairedClientServer(t, func(req Request) Response {
		if req.Method == MethodInitialize {
			return okResponse(req)
		}
		if req.Method == MethodCall {
			return Response{
				JSONRPC: JSONRPCVersion,
				ID:      req.ID,
				Error: &ResponseError{
					Code:    ErrCodeForbidden,
					Message: "forst.node/forbidden",
					Data:    json.RawMessage(`{"moduleId":"legacy/other.ts","exportName":"create"}`),
				},
			}
		}
		return okResponse(req)
	})
	defer func() { _ = server.Close() }()
	if err := client.Initialize(sampleManifest(), nil); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	_, err := client.Call(MethodCall, CallParams{
		ModuleID:   "legacy/other.ts",
		ExportName: "create",
		Args:       json.RawMessage(`[]`),
	})
	if !IsForbidden(err) {
		t.Fatalf("Call forbidden: got %v", err)
	}
}

func TestClient_rpcMethodNotFoundError(t *testing.T) {
	t.Helper()
	client, server := pairedClientServer(t, func(req Request) Response {
		if req.Method == MethodInitialize {
			return okResponse(req)
		}
		return Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Error:   &ResponseError{Code: ErrCodeMethodNotFound, Message: "Method not found"},
		}
	})
	defer func() { _ = server.Close() }()
	if err := client.Initialize(sampleManifest(), nil); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	_, err := client.Call("forst.node/eval", struct{}{})
	if !IsMethodNotFound(err) {
		t.Fatalf("eval method: got %v", err)
	}
}

func TestManifest_AllowCall_rejectsUnknownExport(t *testing.T) {
	t.Helper()
	manifest := sampleManifest()
	err := manifest.AllowCall("legacy/payment.ts", "missing", ExportKindFunction)
	if !IsForbidden(err) {
		t.Fatalf("AllowCall unknown export: got %v", err)
	}
}

func TestManifest_AllowCall_rejectsInvalidModuleID(t *testing.T) {
	t.Helper()
	manifest := sampleManifest()
	cases := []string{
		"../escape.ts",
		"/abs/payment.ts",
		"file://payment.ts",
		"node_modules/pkg/index.ts",
	}
	for _, moduleID := range cases {
		err := manifest.AllowCall(moduleID, "create", ExportKindFunction)
		if err == nil {
			t.Fatalf("AllowCall(%q) expected error", moduleID)
		}
	}
}

func TestManifest_AllowCall_rejectsBeforeRPCSend(t *testing.T) {
	t.Helper()
	client, server := pairedClientServer(t, func(req Request) Response {
		if req.Method == MethodCall {
			t.Fatalf("server should not receive call for forbidden export, got %+v", req)
		}
		return okResponse(req)
	})
	defer func() { _ = server.Close() }()
	if err := client.Initialize(sampleManifest(), nil); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	_, err := client.CallSync("legacy/other.ts", "create", json.RawMessage(`[]`))
	if !IsForbidden(err) {
		t.Fatalf("CallSync preflight: got %v", err)
	}
}

func TestFixtures_matchWireFormat(t *testing.T) {
	t.Helper()
	var req Request
	if err := json.Unmarshal([]byte(readFixture(t, "initialize_request.json")), &req); err != nil {
		t.Fatalf("initialize request fixture: %v", err)
	}
	if req.Method != MethodInitialize {
		t.Fatalf("initialize method = %q", req.Method)
	}
	var resp Response
	if err := json.Unmarshal([]byte(readFixture(t, "forbidden_error.json")), &resp); err != nil {
		t.Fatalf("forbidden fixture: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != ErrCodeForbidden {
		t.Fatalf("forbidden fixture error = %+v", resp.Error)
	}
}

type mockServer struct {
	mu       sync.Mutex
	writeMu  sync.Mutex
	closed   bool
	handler  func(Request) Response
	clientRd *io.PipeReader
	clientWr *io.PipeWriter
	serverRd *io.PipeReader
	serverWr *io.PipeWriter
}

func pairedClientServer(tb testing.TB, handler func(Request) Response) (*Client, *mockServer) {
	tb.Helper()
	clientRd, serverWr := io.Pipe()
	serverRd, clientWr := io.Pipe()
	server := &mockServer{
		handler:  handler,
		clientRd: clientRd,
		clientWr: clientWr,
		serverRd: serverRd,
		serverWr: serverWr,
	}
	client := NewClient(clientRd, clientWr, nil)
	go server.serve(tb)
	return client, server
}

func responseToProtoFrame(resp Response) (*pb.Frame, error) {
	if resp.Error != nil {
		return protoErrorResponse(uint64(resp.ID), int32(resp.Error.Code), resp.Error.Message, resp.Error.Data), nil
	}
	var payload any
	if len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, &payload); err != nil {
			return nil, err
		}
	}
	return protoOKResponse(uint64(resp.ID), payload)
}

func (s *mockServer) serve(tb testing.TB) {
	tb.Helper()
	for {
		frame, err := ReadProtoFrame(s.serverRd, DefaultMaxMsgLen)
		if err != nil {
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()
			if closed {
				return
			}
			if err != io.EOF {
				tb.Errorf("mock server read: %v", err)
			}
			return
		}
		reqBody := frame.GetRequest()
		if reqBody == nil {
			continue
		}
		req := Request{
			JSONRPC: JSONRPCVersion,
			ID:      int64(frame.GetId()),
			Method:  reqBody.Method,
			Params:  reqBody.PayloadJson,
		}
		resp := s.handler(req)
		out, err := responseToProtoFrame(resp)
		if err != nil {
			tb.Errorf("mock server encode: %v", err)
			return
		}
		s.writeMu.Lock()
		err = WriteProtoFrame(s.serverWr, out, DefaultMaxMsgLen)
		s.writeMu.Unlock()
		if err != nil {
			tb.Errorf("mock server write: %v", err)
			return
		}
	}
}

func (s *mockServer) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	_ = s.serverRd.Close()
	_ = s.serverWr.Close()
	_ = s.clientRd.Close()
	_ = s.clientWr.Close()
	return nil
}
