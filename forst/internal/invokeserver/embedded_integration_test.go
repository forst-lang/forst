package invokeserver_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
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
		Host:         "127.0.0.1",
		Port:         "18081",
		Runtime:      "embedded",
		AuthDisabled: true,
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

func TestEmbeddedInvoke_authenticatedHTTP_e2e(t *testing.T) {
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
		Port:    "18082",
		Runtime: "embedded",
	}, backend, invokeserver.DefaultEmbeddedVersion(), nil)
	if err := srv.StartAsync(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Stop() })

	challengeResp, err := http.Get("http://127.0.0.1:18082/invoke/challenge")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = challengeResp.Body.Close() }()
	if challengeResp.StatusCode != http.StatusOK {
		t.Fatalf("challenge status %d", challengeResp.StatusCode)
	}
	var envelope invokeserver.Response
	if err := json.NewDecoder(challengeResp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	var challenge invokeserver.ChallengeResponse
	if err := json.Unmarshal(envelope.Result, &challenge); err != nil {
		t.Fatal(err)
	}
	if challenge.Nonce == "" {
		t.Fatal("expected challenge nonce")
	}

	token, generation := srv.CurrentAuth()
	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:18082/invoke", bytes.NewBufferString(`{"package":"main","function":"Echo","args":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(invokeserver.HeaderInvokeNonce, challenge.Nonce)
	req.Header.Set(invokeserver.HeaderInvokeGeneration, strconv.FormatUint(generation, 10))
	req.Header.Set(invokeserver.HeaderInvokeProof, invokeserver.ComputeInvokeProofForTest(token, generation, challenge.Nonce))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d body=%s", resp.StatusCode, body)
	}
}
