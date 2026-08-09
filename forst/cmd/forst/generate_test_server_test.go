package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/discovery"
	"forst/internal/invokedispatch"
	"forst/internal/invokeserver"
	transformerts "forst/internal/transformer/ts"
)

func TestGenerate_testServer_emitsPromiseSymbolsAndOptionalPeer(t *testing.T) {
	dir := t.TempDir()
	prepareMinimalGenerateProject(t, dir)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	dist := defaultClientDistDir(dir)
	testingDTS := readDistFile(t, dist, "testing.d.ts")
	for _, frag := range []string{
		"startForstTestServer",
		"ForstTestServerOptions",
		"ForstTestServerFailed",
	} {
		if !strings.Contains(testingDTS, frag) {
			t.Fatalf("testing.d.ts missing %q:\n%s", frag, testingDTS)
		}
	}
	pkg := readGeneratedPackageJSON(t, dir)
	peers, _ := pkg["peerDependencies"].(map[string]any)
	if peers["@forst/cli"] != transformerts.CliPeerDependencyRange {
		t.Fatalf("@forst/cli peer = %#v", peers["@forst/cli"])
	}
	meta, _ := pkg["peerDependenciesMeta"].(map[string]any)
	cliMeta, _ := meta["@forst/cli"].(map[string]any)
	if cliMeta["optional"] != true {
		t.Fatalf("expected optional meta:\n%v", pkg)
	}
}

func TestGenerate_testServer_missingPeerThrowsForstTestServerFailed(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not found")
	}
	dir := t.TempDir()
	prepareMinimalGenerateProject(t, dir)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	dist := defaultClientDistDir(dir)
	script := filepath.Join(dist, "missing-cli.mjs")
	body := `
import { startForstTestServer, ForstTestServerFailed } from "./testing.js";
try {
  await startForstTestServer();
  console.error("expected throw");
  process.exit(1);
} catch (e) {
  if (!(e instanceof ForstTestServerFailed)) {
    console.error("wrong error", e);
    process.exit(1);
  }
  if (e.reason !== "cli_missing") {
    console.error("reason", e.reason);
    process.exit(1);
  }
  if (!String(e.installCommand || "").includes("@forst/cli")) {
    console.error("installCommand", e.installCommand);
    process.exit(1);
  }
  console.log("ok");
}
`
	if err := os.WriteFile(script, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("node", script)
	cmd.Dir = dist
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("missing peer script failed: %v\n%s", err, out)
	}
}

func TestGenerate_testServer_stubPeerWiresDefaultClient(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not found")
	}
	dir := t.TempDir()
	ensureNodeModulesDir(t, dir)
	prepareMinimalGenerateProject(t, dir)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	writeStubForstCliInvoke(t, dir, `http://127.0.0.1:9`)

	dist := defaultClientDistDir(dir)
	script := filepath.Join(dist, "stub-peer.mjs")
	body := `
import { startForstTestServer } from "./testing.js";
import { getDefaultInvokeClient, resetDefaultInvokeClientForTest } from "./transport.js";

resetDefaultInvokeClientForTest();
const server = await startForstTestServer({ root: "/tmp/fixture" });
if (server.baseUrl !== "http://127.0.0.1:9") throw new Error("baseUrl " + server.baseUrl);
if (server.connection !== "connect") throw new Error("connection " + server.connection);
const client = getDefaultInvokeClient();
if (!client || typeof client.invokeFunction !== "function") {
  throw new Error("default client missing");
}
// Default client must target the stub base URL (port 9 is unroutable without fetchFn).
await server.stop();
console.log("ok");
`
	if err := os.WriteFile(script, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("node", script)
	cmd.Dir = dist
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stub peer script failed: %v\n%s", err, out)
	}
}

func TestGenerate_testServer_attachPathAgainstInProcessInvoke(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not found")
	}
	dir := t.TempDir()
	ensureNodeModulesDir(t, dir)
	prepareMinimalGenerateProject(t, dir)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}

	reg := invokedispatch.NewRegistry()
	reg.Register(discovery.FunctionInfo{
		Package:  "main",
		Name:     "Echo",
		Runnable: true,
	}, func(args json.RawMessage) (any, error) {
		var in []struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(args, &in); err != nil {
			return nil, err
		}
		msg := ""
		if len(in) > 0 {
			msg = in[0].Message
		}
		return map[string]any{"echo": msg, "timestamp": 42}, nil
	})

	backend := invokeserver.NewRegistryBackend(reg)
	srv := invokeserver.New(
		invokeserver.Config{Host: "127.0.0.1", Port: "0", Runtime: "embedded"},
		backend,
		invokeserver.DefaultEmbeddedVersion(),
		nil,
	)
	if err := srv.StartAsync(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Stop() })
	baseURL := "http://" + srv.BoundAddr()

	// Wait for health.
	deadline := 50
	for i := 0; i < deadline; i++ {
		resp, err := http.Get(baseURL + "/health")
		if err == nil && resp.StatusCode == 200 {
			_ = resp.Body.Close()
			break
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		if i == deadline-1 {
			t.Fatalf("health not ready: %v", err)
		}
	}

	writeStubForstCliInvoke(t, dir, baseURL)

	dist := defaultClientDistDir(dir)
	script := filepath.Join(dist, "attach-e2e.mjs")
	body := fmt.Sprintf(`
import { startForstTestServer } from "./testing.js";
import { Echo } from "./pkg/main.js";

const server = await startForstTestServer({ baseUrl: %q });
try {
  const result = await Echo({ message: "hi" });
  if (result.echo !== "hi" || result.timestamp !== 42) {
    throw new Error("bad result " + JSON.stringify(result));
  }
  console.log("ok");
} finally {
  await server.stop();
}
`, baseURL)
	if err := os.WriteFile(script, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("node", script)
	cmd.Dir = dist
	cmd.Env = append(os.Environ(), "FORST_BASE_URL="+baseURL, "FORST_SKIP_SPAWN=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("attach e2e failed: %v\n%s", err, out)
	}
}

func writeStubForstCliInvoke(t *testing.T, projectRoot, baseURL string) {
	t.Helper()
	stubRoot := filepath.Join(projectRoot, "node_modules", "@forst", "cli")
	if err := os.MkdirAll(filepath.Join(stubRoot, "dist"), 0755); err != nil {
		t.Fatal(err)
	}
	pkg := `{
  "name": "@forst/cli",
  "version": "0.2.0",
  "type": "module",
  "exports": {
    "./invoke": {
      "types": "./dist/invoke.d.ts",
      "import": "./dist/invoke.js",
      "default": "./dist/invoke.js"
    }
  }
}
`
	if err := os.WriteFile(filepath.Join(stubRoot, "package.json"), []byte(pkg), 0644); err != nil {
		t.Fatal(err)
	}
	js := fmt.Sprintf(`
export async function startForstInvokeServer(options = {}) {
  const baseUrl = (options.baseUrl || %q).replace(/\/$/, "");
  const url = new URL(baseUrl);
  return {
    baseUrl,
    port: Number(url.port || 80),
    connection: "connect",
    async stop() {},
    async [Symbol.asyncDispose]() {},
  };
}
`, baseURL)
	if err := os.WriteFile(filepath.Join(stubRoot, "dist", "invoke.js"), []byte(js), 0644); err != nil {
		t.Fatal(err)
	}
	dts := `
export declare function startForstInvokeServer(options?: {
  baseUrl?: string;
}): Promise<{
  baseUrl: string;
  port: number;
  connection: "spawn" | "connect";
  stop(): Promise<void>;
}>;
`
	if err := os.WriteFile(filepath.Join(stubRoot, "dist", "invoke.d.ts"), []byte(dts), 0644); err != nil {
		t.Fatal(err)
	}
}
