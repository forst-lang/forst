package nodert

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	logrus "github.com/sirupsen/logrus"
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

	resetSupervisorForTest()
	t.Cleanup(resetSupervisorForTest)

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	var sawStdoutLog, sawStderrLog bool
	log.Hooks.Add(&testHook{onFire: func(e *logrus.Entry) {
		switch e.Data["event"] {
		case "stdout":
			sawStdoutLog = true
		case "stderr":
			sawStderrLog = true
		}
	}})

	configureBootstrapTestSupervisor(t, bootstrapTestConfig{
		root:     root,
		manifest: manifest,
		log:      log,
	})

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
	if !sawStdoutLog {
		t.Fatal("expected stdout log from bootstrap child")
	}
	if !sawStderrLog {
		t.Fatal("expected stderr log from bootstrap child")
	}
	if err := client.Ping(); err != nil {
		t.Fatalf("Ping after call: %v", err)
	}

	if err := Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

type bootstrapTestConfig struct {
	root          string
	manifest      Manifest
	log           *logrus.Logger
	readyTimeout  time.Duration
	nodePath      string
	bootstrapPath string
}

func configureBootstrapTestSupervisor(t *testing.T, cfg bootstrapTestConfig) {
	t.Helper()

	bootstrap := cfg.bootstrapPath
	if bootstrap == "" {
		var err error
		bootstrap, err = ResolveBootstrapPath(repoRoot(t), "")
		if err != nil {
			t.Skipf("bootstrap not available: %v", err)
		}
	}
	nodePath := cfg.nodePath
	if nodePath == "" {
		var err error
		nodePath, err = ResolveNodeBinary(cfg.root, "node")
		if err != nil {
			t.Fatal(err)
		}
	}
	linkTsxFromRepo(t, cfg.root)

	timeout := cfg.readyTimeout
	if timeout <= 0 {
		timeout = bootstrapReadyTimeout
	}

	ConfigureSupervisor(SupervisorConfig{
		HostReadyTimeout: timeout,
		ProcessOptions: ProcessOptions{
			BoundaryRoot:  cfg.root,
			WorkDir:       cfg.root,
			NodePath:      nodePath,
			BootstrapPath: bootstrap,
			Loader:        "tsx",
			Log:           cfg.log,
		},
		Manifest: cfg.manifest,
	})
}

func longBootstrapBoundaryRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join("/tmp", fmt.Sprintf("forst-long-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	for {
		socketPath, _, err := ResolveBootstrapSocketPath(root)
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(socketPath, filepath.Join("/tmp", "forst-bs-")) {
			t.Cleanup(func() { _ = os.RemoveAll(root) })
			return root
		}
		root = filepath.Join(root, strings.Repeat("p", 40))
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBootstrapSocket_longPathUsesTmpShorteningEndToEnd(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}
	if runtime.GOOS == "windows" {
		t.Skip("bootstrap socket integration uses unix sockets")
	}

	root := longBootstrapBoundaryRoot(t)
	socketPath, _, err := ResolveBootstrapSocketPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(socketPath, filepath.Join("/tmp", "forst-bs-")) {
		t.Fatalf("socket = %q want /tmp/forst-bs-*", socketPath)
	}

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

	resetSupervisorForTest()
	t.Cleanup(resetSupervisorForTest)
	configureBootstrapTestSupervisor(t, bootstrapTestConfig{
		root:     root,
		manifest: manifest,
	})

	client, err := GetClient()
	if err != nil {
		t.Fatalf("GetClient: %v", err)
	}
	if err := client.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if err := Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func TestBootstrapSocket_readyTimeoutCleansSocketFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bootstrap socket integration uses unix sockets")
	}

	root := t.TempDir()
	sleepNode := filepath.Join(root, "sleep-node")
	if err := os.WriteFile(sleepNode, []byte("#!/bin/sh\nsleep 60\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	socketPath, readyPath, err := ResolveBootstrapSocketPath(root)
	if err != nil {
		t.Fatal(err)
	}

	manifest := Manifest{
		Version:      ManifestVersion,
		BoundaryRoot: root,
	}

	resetSupervisorForTest()
	t.Cleanup(resetSupervisorForTest)
	configureBootstrapTestSupervisor(t, bootstrapTestConfig{
		root:         root,
		manifest:     manifest,
		nodePath:     sleepNode,
		readyTimeout: 200 * time.Millisecond,
	})

	_, err = GetClient()
	if err == nil {
		_ = Shutdown()
		t.Fatal("expected bootstrap ready timeout")
	}
	if !strings.Contains(err.Error(), "host ready timeout") {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(socketPath); err == nil {
		t.Fatalf("expected socket file removed: %s", socketPath)
	}
	if _, err := os.Stat(readyPath); err == nil {
		t.Fatalf("expected ready file removed: %s", readyPath)
	}
}

func TestBootstrapSocket_spawnFailureCleansSocketFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bootstrap socket integration uses unix sockets")
	}

	root := t.TempDir()
	socketPath, readyPath, err := ResolveBootstrapSocketPath(root)
	if err != nil {
		t.Fatal(err)
	}

	manifest := Manifest{
		Version:      ManifestVersion,
		BoundaryRoot: root,
	}

	resetSupervisorForTest()
	t.Cleanup(resetSupervisorForTest)
	configureBootstrapTestSupervisor(t, bootstrapTestConfig{
		root:     root,
		manifest: manifest,
		nodePath: filepath.Join(root, "missing-node-binary"),
	})

	_, err = GetClient()
	if err == nil {
		_ = Shutdown()
		t.Fatal("expected spawn failure")
	}
	if _, err := os.Stat(socketPath); err == nil {
		t.Fatalf("expected socket file removed after spawn failure: %s", socketPath)
	}
	if _, err := os.Stat(readyPath); err == nil {
		t.Fatalf("expected ready file removed after spawn failure: %s", readyPath)
	}
}
