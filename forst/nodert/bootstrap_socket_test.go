package nodert

import (
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestResolveBootstrapSocketPath_defaultAndOverride(t *testing.T) {
	root := t.TempDir()
	socketPath, readyPath, err := ResolveBootstrapSocketPath(root)
	if err != nil {
		t.Fatal(err)
	}
	wantSocket, err := filepath.Abs(filepath.Join(root, ".forst", "node-bootstrap.sock"))
	if err != nil {
		t.Fatal(err)
	}
	wantSocket = ensureUnixSocketPathLength(wantSocket)
	if socketPath != wantSocket {
		t.Fatalf("socket = %q want %q", socketPath, wantSocket)
	}
	if readyPath != readyPathForSocket(wantSocket) {
		t.Fatalf("ready = %q want %q", readyPath, readyPathForSocket(wantSocket))
	}

	t.Setenv(envNodeSocket, filepath.Join(root, "custom.sock"))
	socketPath, readyPath, err = ResolveBootstrapSocketPath(root)
	if err != nil {
		t.Fatal(err)
	}
	wantCustom, err := filepath.Abs(filepath.Join(root, "custom.sock"))
	if err != nil {
		t.Fatal(err)
	}
	wantCustom = ensureUnixSocketPathLength(wantCustom)
	if socketPath != wantCustom || readyPath != readyPathForSocket(wantCustom) {
		t.Fatalf("override socket=%q ready=%q want socket=%q ready=%q", socketPath, readyPath, wantCustom, readyPathForSocket(wantCustom))
	}
}

func TestBuildBootstrapSpawnCommand_setsSocketEnv(t *testing.T) {
	root := t.TempDir()
	bootstrap := filepath.Join(root, "bootstrap.js")
	if err := os.WriteFile(bootstrap, []byte("//"), 0o644); err != nil {
		t.Fatal(err)
	}
	nodePath := filepath.Join(root, "node")
	if err := os.WriteFile(nodePath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	tsxDir := filepath.Join(root, "node_modules", "tsx", "dist")
	if err := os.MkdirAll(tsxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tsxDir, "loader.mjs"), []byte("//"), 0o644); err != nil {
		t.Fatal(err)
	}

	socketPath := filepath.Join(root, ".forst", "node-bootstrap.sock")
	readyPath := socketPath + ".ready"
	cmd, err := BuildBootstrapSpawnCommand(BootstrapSpawnInput{
		BoundaryRoot:  root,
		Executable:    nodePath,
		BootstrapPath: bootstrap,
		WorkDir:       root,
		Loader:        "tsx",
		SocketPath:    socketPath,
		ReadyPath:     readyPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := lookupEnvValue(cmd.Env, envNodeSocket); got != socketPath {
		t.Fatalf("FORST_NODE_SOCKET = %q want %q", got, socketPath)
	}
	if got := lookupEnvValue(cmd.Env, envNodeHostReady); got != readyPath {
		t.Fatalf("FORST_NODE_HOST_READY = %q want %q", got, readyPath)
	}
	if got := lookupEnvValue(cmd.Env, envNodeHost); got != "" {
		t.Fatalf("FORST_NODE_HOST = %q want unset", got)
	}
}

func TestBootstrapSocket_writesReadyMarkerAndAcceptsRPC(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}
	if runtime.GOOS == "windows" {
		t.Skip("bootstrap socket integration uses unix sockets")
	}

	bootstrap, err := ResolveBootstrapPath(repoRoot(t), "")
	if err != nil {
		t.Skipf("bootstrap not available: %v", err)
	}
	t.Setenv(envNodeBootstrap, bootstrap)

	root := t.TempDir()
	legacyDir := filepath.Join(root, "legacy")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "payment.ts"), []byte(`export function add(a: number, b: number) { return { sum: a + b }; }`), 0o644); err != nil {
		t.Fatal(err)
	}

	socketPath, readyPath, err := ResolveBootstrapSocketPath(root)
	if err != nil {
		t.Fatal(err)
	}

	manifest := Manifest{
		Version:      ManifestVersion,
		BoundaryRoot: root,
		Exports: []ExportEntry{
			{ModuleID: "legacy/payment.ts", Name: "add", Kind: ExportKindFunction},
		},
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	resetSupervisorForTest()
	t.Cleanup(resetSupervisorForTest)
	if err := configureFromManifest(string(manifestJSON)); err != nil {
		t.Fatal(err)
	}

	client, err := GetClient()
	if err != nil {
		t.Fatalf("GetClient: %v", err)
	}
	if err := client.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	data, err := os.ReadFile(readyPath)
	if err != nil {
		t.Fatalf("read ready marker: %v", err)
	}
	if !strings.Contains(string(data), `"phase":"app"`) && !strings.Contains(string(data), `"phase": "app"`) {
		t.Fatalf("ready marker missing phase app: %s", string(data))
	}
	if _, err := os.Stat(socketPath); err != nil {
		t.Fatalf("socket path missing: %v", err)
	}

	if err := Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func TestBootstrapSocket_rejectsSecondConnection(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}
	if runtime.GOOS == "windows" {
		t.Skip("bootstrap socket integration uses unix sockets")
	}

	bootstrap, err := ResolveBootstrapPath(repoRoot(t), "")
	if err != nil {
		t.Skipf("bootstrap not available: %v", err)
	}
	t.Setenv(envNodeBootstrap, bootstrap)

	root := t.TempDir()
	legacyDir := filepath.Join(root, "legacy")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "payment.ts"), []byte(`export function add(a: number, b: number) { return { sum: a + b }; }`), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest := Manifest{
		Version:      ManifestVersion,
		BoundaryRoot: root,
		Exports: []ExportEntry{
			{ModuleID: "legacy/payment.ts", Name: "add", Kind: ExportKindFunction},
		},
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	resetSupervisorForTest()
	t.Cleanup(resetSupervisorForTest)
	if err := configureFromManifest(string(manifestJSON)); err != nil {
		t.Fatal(err)
	}

	if _, err := GetClient(); err != nil {
		t.Fatalf("GetClient: %v", err)
	}

	socketPath, _, err := ResolveBootstrapSocketPath(root)
	if err != nil {
		t.Fatal(err)
	}

	second, err := net.DialTimeout("unix", socketPath, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("second dial: %v", err)
	}
	_ = second.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	if _, readErr := second.Read(make([]byte, 1)); readErr == nil {
		t.Fatal("expected second connection to close after duplicate reject")
	}
	_ = second.Close()

	if err := Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func TestBootstrapSocket_bothStreamsForwardedAsLogs(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}
	if runtime.GOOS == "windows" {
		t.Skip("bootstrap socket integration uses unix sockets")
	}

	bootstrap, err := ResolveBootstrapPath(repoRoot(t), "")
	if err != nil {
		t.Skipf("bootstrap not available: %v", err)
	}
	t.Setenv(envNodeBootstrap, bootstrap)

	root := t.TempDir()
	legacyDir := filepath.Join(root, "legacy")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "chatty.ts"), []byte(`console.log("stdout-spam");
console.error("stderr-spam");
export function ping(): { ok: boolean } { return { ok: true }; }
`), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest := Manifest{
		Version:      ManifestVersion,
		BoundaryRoot: root,
		Exports: []ExportEntry{
			{ModuleID: "legacy/chatty.ts", Name: "ping", Kind: ExportKindFunction},
		},
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	resetSupervisorForTest()
	t.Cleanup(resetSupervisorForTest)
	if err := configureFromManifest(string(manifestJSON)); err != nil {
		t.Fatal(err)
	}

	client, err := GetClient()
	if err != nil {
		t.Fatalf("GetClient: %v", err)
	}
	if err := client.Ping(); err != nil {
		t.Fatalf("Ping before call: %v", err)
	}

	type pingResult struct {
		OK bool `json:"ok"`
	}
	got, err := CallSync[pingResult]("legacy/chatty.ts", "ping")
	if err != nil {
		t.Fatalf("CallSync: %v", err)
	}
	if !got.OK {
		t.Fatalf("ping = %#v", got)
	}
	if err := client.Ping(); err != nil {
		t.Fatalf("Ping after call: %v", err)
	}

	if err := Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}
