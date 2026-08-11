package invokeserver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestWriteAuthArtifacts_envDeliveryNoTokenFile(t *testing.T) {
	workDir := t.TempDir()
	t.Setenv(envInvokeToken, "")
	t.Setenv(EnvInvokeAuthFD, "")

	cfg := Config{Host: "127.0.0.1", Port: "0", Runtime: "embedded", BoundaryRoot: workDir}
	ApplyListenDefaults(&cfg, workDir)
	s := New(cfg, &stubBackend{}, DefaultEmbeddedVersion(), nil)

	if err := s.WriteAuthArtifacts(workDir, cfg); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(workDir, ".forst", "invoke.ready"))
	if err != nil {
		t.Fatal(err)
	}
	var payload InvokeReadyPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.TokenDelivery != tokenDeliveryEnv {
		t.Fatalf("tokenDelivery = %q, want %q", payload.TokenDelivery, tokenDeliveryEnv)
	}
	if os.Getenv(envInvokeToken) == "" {
		t.Fatal("expected FORST_INVOKE_TOKEN in environment")
	}
	if _, err := os.Stat(filepath.Join(workDir, ".forst", "invoke.token")); !os.IsNotExist(err) {
		t.Fatalf("invoke.token should not exist, err=%v", err)
	}
}

func TestWriteAuthArtifacts_handoffDelivery(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	t.Setenv(EnvInvokeAuthFD, strconv.Itoa(int(w.Fd())))
	t.Setenv(envInvokeToken, "")

	workDir := t.TempDir()
	cfg := Config{Host: "127.0.0.1", Port: "0", Runtime: "embedded", BoundaryRoot: workDir}
	ApplyListenDefaults(&cfg, workDir)
	s := New(cfg, &stubBackend{}, DefaultEmbeddedVersion(), nil)

	if err := s.WriteAuthArtifacts(workDir, cfg); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(workDir, ".forst", "invoke.ready"))
	if err != nil {
		t.Fatal(err)
	}
	var payload InvokeReadyPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.TokenDelivery != tokenDeliveryHandoff {
		t.Fatalf("tokenDelivery = %q, want %q", payload.TokenDelivery, tokenDeliveryHandoff)
	}
	if _, err := os.Stat(filepath.Join(workDir, ".forst", "invoke.token")); !os.IsNotExist(err) {
		t.Fatalf("invoke.token should not exist, err=%v", err)
	}

	gen, token, err := readAuthHandoff(r)
	if err != nil {
		t.Fatal(err)
	}
	if gen == 0 || len(token) == 0 {
		t.Fatalf("handoff gen=%d tokenLen=%d", gen, len(token))
	}
	serverToken, serverGen := s.CurrentAuth()
	if gen != serverGen {
		t.Fatalf("handoff generation = %d, want %d", gen, serverGen)
	}
	if string(token) != string(serverToken) {
		t.Fatal("handoff token mismatch")
	}
}

func TestLogAuthDisabledWarning_alwaysLogs(t *testing.T) {
	log := &captureLogger{}
	s := New(Config{
		Host:         "127.0.0.1",
		Port:         "0",
		Runtime:      "embedded",
		AuthDisabled: true,
	}, &stubBackend{}, DefaultEmbeddedVersion(), log)

	s.logAuthDisabledWarning()
	if len(log.infos) != 1 {
		t.Fatalf("expected one warning log, got %d: %v", len(log.infos), log.infos)
	}
	if log.infos[0] == "" {
		t.Fatal("expected non-empty warning")
	}
}
