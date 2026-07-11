package nodert

import (
	"encoding/json"
	"testing"
)

func BenchmarkClientGenNextBatch_singleStep(b *testing.B) {
	client, server := pairedClientServer(b, func(req Request) Response {
		if req.Method == MethodInitialize {
			return okResponse(req)
		}
		if req.Method == MethodGenNextBatch {
			return Response{
				JSONRPC: JSONRPCVersion,
				ID:      req.ID,
				Result:  json.RawMessage(`{"steps":[{"kind":"yield","value":1}]}`),
			}
		}
		return okResponse(req)
	})
	defer func() { _ = server.Close() }()
	if err := client.Initialize(sampleManifest(), nil); err != nil {
		b.Fatal(err)
	}
	streamID := "bench-stream"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = clientGenNextBatch[int](client, streamID, 1)
	}
}

func BenchmarkClientGenNextBatch_batch32(b *testing.B) {
	client, server := pairedClientServer(b, func(req Request) Response {
		if req.Method == MethodInitialize {
			return okResponse(req)
		}
		if req.Method == MethodGenNextBatch {
			steps := make([]map[string]any, 32)
			for i := range steps {
				steps[i] = map[string]any{"kind": "yield", "value": i}
			}
			raw, _ := json.Marshal(map[string]any{"steps": steps})
			return Response{JSONRPC: JSONRPCVersion, ID: req.ID, Result: raw}
		}
		return okResponse(req)
	})
	defer func() { _ = server.Close() }()
	if err := client.Initialize(sampleManifest(), nil); err != nil {
		b.Fatal(err)
	}
	streamID := "bench-stream"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = clientGenNextBatch[int](client, streamID, 32)
	}
}
