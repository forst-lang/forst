package nodert

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHostMode_sharedSingletonVisibleToRPC(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}
	if runtime.GOOS == "windows" {
		t.Skip("host integration uses unix sockets")
	}

	root, _ := setupHostIntegrationRoot(t, hostServerWithSingleton)
	manifest := hostCounterManifest(root)
	runHostCounterRPC(t, root, manifest, 2, 3)
}

func TestHostMode_stdoutSpam_rpcStable(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}
	if runtime.GOOS == "windows" {
		t.Skip("host integration uses unix sockets")
	}

	root, _ := setupHostIntegrationRoot(t, hostServerWithStdoutSpam)
	manifest := hostCounterManifest(root)
	runHostCounterRPC(t, root, manifest, 1, 2)
}

func TestHostMode_remixServe_sharedTodosVisibleToRPC(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}
	if runtime.GOOS == "windows" {
		t.Skip("host integration uses unix sockets")
	}

	root := setupRemixServeIntegrationRoot(t)
	manifest := remixTodosManifest(root)
	runHostEditCountRPC(t, root, manifest, 2, 3)
}

func TestHostMode_hostNotStarted_timesOut(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}
	if runtime.GOOS == "windows" {
		t.Skip("host integration uses unix sockets")
	}

	root, _ := setupHostIntegrationRoot(t, hostExitsImmediately)
	manifest := hostCounterManifest(root)

	resetSupervisorForTest()
	t.Setenv(envNodeBootstrap, "")
	t.Setenv(envNodeBinary, "")
	t.Setenv(envNodeSocket, shortHostSocketPath(t))

	hostAutoRegister := false
	writeHostFtconfig(t, root, []string{"app/server.mjs"}, 1, &hostAutoRegister)
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := configureFromManifest(string(manifestJSON)); err != nil {
		t.Fatal(err)
	}

	_, err = GetClient()
	if err == nil {
		_ = Shutdown()
		t.Fatal("expected host ready timeout")
	}
	if !strings.Contains(err.Error(), "host ready timeout") {
		t.Fatalf("err = %v", err)
	}
}

type hostSocketFixture struct {
	dir  string
	once sync.Once
}

var hostSocketFixtures sync.Map

func shortHostSocketPath(t *testing.T) string {
	t.Helper()
	fixAny, _ := hostSocketFixtures.LoadOrStore(t, &hostSocketFixture{})
	fix := fixAny.(*hostSocketFixture)
	fix.once.Do(func() {
		// Keep the full path under the Unix socket length limit (104 bytes on macOS).
		fix.dir = filepath.Join("/tmp", fmt.Sprintf("fh-%d", time.Now().UnixNano()))
		if err := os.MkdirAll(fix.dir, 0o750); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			hostSocketFixtures.Delete(t)
			_ = os.RemoveAll(fix.dir)
		})
	})
	return filepath.Join(fix.dir, "s.sock")
}

func linkTsxFromRepo(t *testing.T, root string) {
	t.Helper()
	repo := repoRoot(t)
	src := filepath.Join(repo, "node_modules", "tsx")
	if _, err := os.Stat(src); err != nil {
		t.Skipf("tsx not installed in monorepo: %v", err)
	}
	dstDir := filepath.Join(root, "node_modules")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dstDir, "tsx")
	if err := os.Symlink(src, dst); err != nil {
		t.Fatal(err)
	}
}

func hostCounterManifest(root string) Manifest {
	return Manifest{
		Version:      ManifestVersion,
		BoundaryRoot: root,
		Exports: []ExportEntry{
			{ModuleID: "legacy/counter.ts", Name: "inc", Kind: ExportKindFunction},
		},
	}
}

func remixTodosManifest(root string) Manifest {
	return Manifest{
		Version:      ManifestVersion,
		BoundaryRoot: root,
		Exports: []ExportEntry{
			{ModuleID: "legacy/todos.ts", Name: "bumpEditCount", Kind: ExportKindFunction},
		},
	}
}

func setupRemixServeIntegrationRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join(repoRoot(t), "examples", "in", "rfc", "node-interop", "remix-serve"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "node_modules", ".bin", "remix-serve")); err != nil {
		t.Skipf("remix-serve example deps not installed (bun install in example dir): %v", err)
	}
	t.Setenv(envNodeSocket, shortHostSocketPath(t))
	t.Setenv("HOST", "127.0.0.1")
	t.Cleanup(func() {
		_ = os.RemoveAll(filepath.Join(root, ".forst"))
	})
	return root
}

func linkNodeRuntimeFromRepo(t *testing.T, root string) {
	t.Helper()
	repo := repoRoot(t)
	src := filepath.Join(repo, "packages", "node-runtime")
	if _, err := os.Stat(filepath.Join(src, "dist", "host.js")); err != nil {
		t.Skipf("node-runtime not built (run task build:node-runtime): %v", err)
	}
	dstDir := filepath.Join(root, "node_modules", "@forst")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dstDir, "node-runtime")
	if err := os.Symlink(src, dst); err != nil {
		t.Fatal(err)
	}
}

func setupHostIntegrationRoot(t *testing.T, serverBody func(hostJS string) string) (string, string) {
	t.Helper()
	root := t.TempDir()
	hostJS := hostModulePath(t)

	legacyDir := filepath.Join(root, "legacy")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	counterTS := `declare const globalThis: { __forstTest?: { n: number } };
export function inc(): number {
  if (!globalThis.__forstTest) {
    globalThis.__forstTest = { n: 0 };
  }
  return ++globalThis.__forstTest.n;
}
`
	if err := os.WriteFile(filepath.Join(legacyDir, "counter.ts"), []byte(counterTS), 0o644); err != nil {
		t.Fatal(err)
	}

	appDir := filepath.Join(root, "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	server := serverBody(hostJS)
	if err := os.WriteFile(filepath.Join(appDir, "server.mjs"), []byte(server), 0o644); err != nil {
		t.Fatal(err)
	}

	linkTsxFromRepo(t, root)
	linkNodeRuntimeFromRepo(t, root)
	t.Setenv(envNodeSocket, shortHostSocketPath(t))

	writeHostFtconfig(t, root, []string{"app/server.mjs"}, 30)
	return root, hostJS
}

func hostServerWithSingleton(hostJS string) string {
	return fmt.Sprintf(`import { signalForstAppReady } from %q;

globalThis.__forstTest = { n: 1 };
await signalForstAppReady();
`, fileURL(hostJS))
}

func hostServerWithStdoutSpam(hostJS string) string {
	return fmt.Sprintf(`import { signalForstAppReady } from %q;

globalThis.__forstTest = { n: 0 };
await signalForstAppReady();
setInterval(() => { console.log("host-spam"); }, 50);
`, fileURL(hostJS))
}

func hostExitsImmediately(hostJS string) string {
	_ = hostJS
	return "// exits immediately — no host, no keepalive\n"
}

func fileURL(path string) string {
	return "file://" + filepath.ToSlash(path)
}

func hostModulePath(t *testing.T) string {
	t.Helper()
	path := filepath.Join(repoRoot(t), "packages", "node-runtime", "dist", "host.js")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("host.js not built (run task build:node-runtime): %v", err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func writeHostFtconfig(t *testing.T, root string, args []string, readyTimeoutSec int, hostAutoRegister ...*bool) {
	t.Helper()
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	hostAutoRegisterJSON := ""
	if len(hostAutoRegister) > 0 && hostAutoRegister[0] != nil {
		hostAutoRegisterJSON = fmt.Sprintf("\n    \"hostAutoRegister\": %t,", *hostAutoRegister[0])
	}
	cfg := fmt.Sprintf(`{
  "files": {
    "include": ["**/*.ft", "**/*.ts"],
    "exclude": ["**/node_modules/**"]
  },
  "node": {
    "enabled": true,
    "runtimeEnabled": true,
    "hostMode": true,%s
    "args": %s,
    "hostReadyTimeoutSeconds": %d
  }
}`, hostAutoRegisterJSON, string(argsJSON), readyTimeoutSec)
	if err := os.WriteFile(filepath.Join(root, "ftconfig.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runHostCounterRPC(t *testing.T, _ string, manifest Manifest, firstWant, secondWant int) {
	t.Helper()

	resetSupervisorForTest()
	t.Cleanup(resetSupervisorForTest)
	t.Setenv(envNodeBootstrap, "")
	t.Setenv(envNodeBinary, "")

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := configureFromManifest(string(manifestJSON)); err != nil {
		t.Fatal(err)
	}

	got1, err := CallSync[float64]("legacy/counter.ts", "inc")
	if err != nil {
		t.Fatalf("first CallSync: %v", err)
	}
	if int(got1) != firstWant {
		t.Fatalf("first inc = %v want %d", got1, firstWant)
	}

	got2, err := CallSync[float64]("legacy/counter.ts", "inc")
	if err != nil {
		t.Fatalf("second CallSync: %v", err)
	}
	if int(got2) != secondWant {
		t.Fatalf("second inc = %v want %d", got2, secondWant)
	}

	if err := Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func runHostEditCountRPC(t *testing.T, _ string, manifest Manifest, firstWant, secondWant int) {
	t.Helper()

	resetSupervisorForTest()
	t.Cleanup(resetSupervisorForTest)
	t.Setenv(envNodeBootstrap, "")
	t.Setenv(envNodeBinary, "")

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := configureFromManifest(string(manifestJSON)); err != nil {
		t.Fatal(err)
	}

	got1, err := CallSync[float64]("legacy/todos.ts", "bumpEditCount")
	if err != nil {
		t.Fatalf("first CallSync: %v", err)
	}
	if int(got1) != firstWant {
		t.Fatalf("first bumpEditCount = %v want %d", got1, firstWant)
	}

	got2, err := CallSync[float64]("legacy/todos.ts", "bumpEditCount")
	if err != nil {
		t.Fatalf("second CallSync: %v", err)
	}
	if int(got2) != secondWant {
		t.Fatalf("second bumpEditCount = %v want %d", got2, secondWant)
	}

	if err := Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}
