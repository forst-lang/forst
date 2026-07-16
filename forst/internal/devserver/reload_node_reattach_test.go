package devserver

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func setupReloadHostFixture(t *testing.T) string {
	t.Helper()
	repo := reloadRepoRoot(t)
	hostJS := filepath.Join(repo, "packages", "node-runtime", "dist", "host.js")
	if _, err := os.Stat(hostJS); err != nil {
		t.Skipf("host.js not built: %v", err)
	}
	hostJS, err := filepath.Abs(hostJS)
	if err != nil {
		t.Fatal(err)
	}

	root := t.TempDir()
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
	server := fmt.Sprintf(`import { signalForstAppReady } from %q;

globalThis.__forstTest = { n: 1 };
await signalForstAppReady();
`, "file://"+filepath.ToSlash(hostJS))
	if err := os.WriteFile(filepath.Join(appDir, "server.mjs"), []byte(server), 0o644); err != nil {
		t.Fatal(err)
	}

	tsxSrc := filepath.Join(repo, "node_modules", "tsx")
	if _, err := os.Stat(tsxSrc); err != nil {
		t.Skipf("tsx not installed: %v", err)
	}
	nodeRTSrc := filepath.Join(repo, "packages", "node-runtime")
	if _, err := os.Stat(filepath.Join(nodeRTSrc, "dist", "host.js")); err != nil {
		t.Skipf("node-runtime not built: %v", err)
	}
	for _, link := range []struct{ dir, name, src string }{
		{filepath.Join(root, "node_modules"), "tsx", tsxSrc},
		{filepath.Join(root, "node_modules", "@forst"), "node-runtime", nodeRTSrc},
	} {
		if err := os.MkdirAll(link.dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(link.src, filepath.Join(link.dir, link.name)); err != nil {
			t.Fatal(err)
		}
	}

	sockDir := filepath.Join(root, ".forst")
	if err := os.MkdirAll(sockDir, 0o750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FORST_NODE_SOCKET", filepath.Join(sockDir, "node.sock"))

	argsJSON, _ := json.Marshal([]string{"app/server.mjs"})
	cfg := fmt.Sprintf(`{
  "files": {"include": ["**/*.ft", "**/*.ts"], "exclude": ["**/node_modules/**"]},
  "node": {
    "enabled": true,
    "runtimeEnabled": true,
    "hostMode": true,
    "args": %s,
    "hostReadyTimeoutSeconds": 30
  }
}`, string(argsJSON))
	if err := os.WriteFile(filepath.Join(root, "ftconfig.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func reloadRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.work not found")
		}
		dir = parent
	}
}

func TestReload_nodeHostSurvivesHostModeStopAndReattaches(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}
	if runtime.GOOS == "windows" {
		t.Skip("unix host sockets")
	}
	TestReload_parentOwnedHostSurvivesGroupKill(t)
}
