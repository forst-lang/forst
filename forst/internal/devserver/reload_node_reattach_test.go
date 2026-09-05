package devserver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func setupReloadHostFixture(t *testing.T) string {
	t.Helper()
	repo := reloadRepoRoot(t)
	hostJS := filepath.Join(repo, "packages", "runtime", "dist", "host.js")
	if _, err := os.Stat(hostJS); err != nil {
		t.Skipf("host.js not built: %v", err)
	}
	hostJS, err := filepath.Abs(hostJS)
	if err != nil {
		t.Fatal(err)
	}

	root := t.TempDir()
	counterJS := `export function inc() {
  if (!globalThis.__forstTest) {
    globalThis.__forstTest = { n: 0 };
  }
  return ++globalThis.__forstTest.n;
}
`
	compiledDir := filepath.Join(root, ".forst", "js", "legacy")
	if err := os.MkdirAll(compiledDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(compiledDir, "counter.js"), []byte(counterJS), 0o644); err != nil {
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
	nodeRTSrc := filepath.Join(repo, "packages", "runtime")
	if _, err := os.Stat(filepath.Join(nodeRTSrc, "dist", "host.js")); err != nil {
		t.Skipf("/runtime not built: %v", err)
	}
	for _, link := range []struct{ dir, name, src string }{
		{filepath.Join(root, "node_modules"), "tsx", tsxSrc},
		{filepath.Join(root, "node_modules", "@forst"), "runtime", nodeRTSrc},
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
	t.Setenv("FORST_BRIDGE_SOCKET", filepath.Join(sockDir, "node.sock"))

	argsJSON, _ := json.Marshal([]string{"app/server.mjs"})
	cfg := fmt.Sprintf(`{
  "files": {"include": ["**/*.ft", "**/*.ts"], "exclude": ["**/node_modules/**"]},
  "bridge": {
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
