package invokeserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"forst/internal/discovery"
	"forst/internal/invokedispatch"
	"forst/internal/invokeserver"
)

func TestEmbeddedInvoke_e2e(t *testing.T) {
	reg := invokedispatch.NewRegistry()
	reg.Register(discovery.FunctionInfo{
		Package:  "main",
		Name:     "Echo",
		Runnable: true,
	}, func(_ json.RawMessage) (any, error) {
		return map[string]any{"echo": "hi"}, nil
	})

	backend := invokeserver.NewRegistryBackend(reg)
	srv := invokeserver.New(invokeserver.Config{
		Host:    "127.0.0.1",
		Port:    "18081",
		Runtime: "embedded",
	}, backend, invokeserver.DefaultEmbeddedVersion(), nil)
	if err := srv.StartAsync(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Stop() })

	body := bytes.NewBufferString(`{"package":"main","function":"Echo","args":[]}`)
	resp, err := http.Post("http://127.0.0.1:18081/invoke", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}
