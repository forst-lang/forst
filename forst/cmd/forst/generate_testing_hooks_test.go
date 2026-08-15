package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ftconfig"
	transformerts "forst/internal/transformer/ts"
)

func TestGenerate_testingSubpathInExportsMap(t *testing.T) {
	j := generateClientPackageJSON(ftconfig.EffectiveGenerateConfig(nil, ""), []string{"auth"}, nil)
	var pkg map[string]any
	if err := json.Unmarshal([]byte(j), &pkg); err != nil {
		t.Fatal(err)
	}
	exports := pkg["exports"].(map[string]any)
	entry, ok := exports["./$testing"].(map[string]any)
	if !ok {
		t.Fatalf("missing ./$testing export:\n%s", j)
	}
	if entry["types"] != "./dist/$testing.d.ts" {
		t.Fatalf("types = %#v", entry["types"])
	}
	if entry["default"] != "./dist/$testing.js" {
		t.Fatalf("default = %#v", entry["default"])
	}
}

func TestGenerate_testingSubpathHonoursConfig(t *testing.T) {
	cfg := ftconfig.EffectiveGenerateConfig(nil, "")
	cfg.TestingSubpath = "$test-double"
	j := generateClientPackageJSON(cfg, []string{"main"}, nil)
	var pkg map[string]any
	if err := json.Unmarshal([]byte(j), &pkg); err != nil {
		t.Fatal(err)
	}
	exports := pkg["exports"].(map[string]any)
	if _, ok := exports["./$testing"]; ok {
		t.Fatal("default ./$testing must not appear when testingSubpath overridden")
	}
	entry := exports["./$test-double"].(map[string]any)
	if entry["default"] != "./dist/$test-double.js" {
		t.Fatalf("default = %#v", entry["default"])
	}
}

func TestGenerate_testingModuleEmitsOverrideTypesPerPackage(t *testing.T) {
	dir := t.TempDir()
	prepareMinimalGenerateProject(t, dir)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	dts, err := os.ReadFile(filepath.Join(defaultClientDistDir(dir), "$testing.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(dts)
	for _, frag := range []string{
		"export type MainHandlers",
		"Echo:",
		"export interface ForstTestOverrides",
		"packages?:",
		"main?: Partial<MainHandlers>",
		"client?: Partial<ForstInvokeClient>",
		"withForstTestScope",
	} {
		if !strings.Contains(got, frag) {
			t.Fatalf("missing %q in testing.d.ts:\n%s", frag, got)
		}
	}
}

func TestGenerate_testingOverridesKeyedUnderPackagesNotAtTopLevel(t *testing.T) {
	got := transformerts.EmitTestingDTS([]transformerts.ModuleEmit{{
		PackageName: "auth",
		Functions: []transformerts.FunctionSignature{{
			Name:       "VerifyToken",
			Parameters: []transformerts.Parameter{{Name: "input", Type: "$VerifyTokenRequest"}},
			ReturnType: "$VerifyTokenResponse",
		}},
		TypeImports: []string{"$VerifyTokenRequest", "$VerifyTokenResponse"},
	}}, "@forst/gen", transformerts.RuntimePromise)
	if !strings.Contains(got, "packages?:") {
		t.Fatal("missing packages key")
	}
	ifaceStart := strings.Index(got, "export interface ForstTestOverrides")
	fnStart := strings.Index(got, "export declare function withForstTestScope")
	if ifaceStart < 0 || fnStart < 0 {
		t.Fatalf("missing ForstTestOverrides or withForstTestScope in:\n%s", got)
	}
	body := got[ifaceStart:fnStart]
	if strings.Contains(body, "\n  auth?:") {
		t.Fatalf("auth must be under packages, not top-level:\n%s", body)
	}
	if !strings.Contains(body, "auth?: Partial<AuthHandlers>") {
		t.Fatalf("missing packages.auth:\n%s", body)
	}
}

func TestGenerate_transportEmitsMiddlewareTypes(t *testing.T) {
	got := transformerts.EmitTransportDTS()
	for _, frag := range []string{
		"ForstInvokeMiddleware",
		"InvokeContext",
		"onStart?",
		"onSuccess?",
		"onFailure?",
		"middleware?: ForstInvokeMiddleware[]",
	} {
		if !strings.Contains(got, frag) {
			t.Fatalf("missing %q", frag)
		}
	}
}

func TestGenerate_middlewareContextIncludesPackageAndFunction(t *testing.T) {
	got := transformerts.EmitTransportESM("6321", transformerts.RuntimePromise, nil)
	for _, frag := range []string{
		"packageName",
		"functionName",
		"attempt:",
		"startedAt",
		"onStart",
		"onSuccess",
		"onFailure",
		"this.middleware",
		"configureDefaultInvokeClient",
	} {
		if !strings.Contains(got, frag) {
			t.Fatalf("missing %q in transport ESM", frag)
		}
	}
}

func TestGenerate_emitsTestingModuleFiles(t *testing.T) {
	dir := t.TempDir()
	prepareMinimalGenerateProject(t, dir)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	dist := defaultClientDistDir(dir)
	for _, rel := range []string{"$testing.js", "$testing.d.ts"} {
		if _, err := os.Stat(filepath.Join(dist, rel)); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}
}

func TestGenerate_acceptance_scopedFunctionOverride(t *testing.T) {
	runPhase5NodeAcceptance(t, "scoped-function.mjs", `
import { withForstTestScope } from "./$testing.js";
import { $main } from "./pkg/main.js";

const result = await withForstTestScope(
  {
    packages: {
      main: {
        Echo: async () => ({ echo: "fake", timestamp: 1 }),
      },
    },
  },
  async () => $main.Echo({ message: "prod-import" })
);
if (result.echo !== "fake") {
  throw new Error("expected fake override, got " + JSON.stringify(result));
}
console.log("ok");
`)
}

func TestGenerate_acceptance_scopedPackageOverride(t *testing.T) {
	runPhase5NodeAcceptance(t, "scoped-package.mjs", `
import { withForstTestScope } from "./$testing.js";
import { $main } from "./pkg/main.js";

const result = await withForstTestScope(
  {
    packages: {
      main: {
        Echo: async (input) => ({ echo: "pkg:" + input.message, timestamp: 2 }),
      },
    },
  },
  async () => $main.Echo({ message: "hi" })
);
if (result.echo !== "pkg:hi") {
  throw new Error("package override failed: " + JSON.stringify(result));
}
console.log("ok");
`)
}

func TestGenerate_acceptance_scopedTransportOverride(t *testing.T) {
	runPhase5NodeAcceptance(t, "scoped-transport.mjs", `
import { withForstTestScope } from "./$testing.js";
import { $main } from "./pkg/main.js";

let seen;
const result = await withForstTestScope(
  {
    client: {
      async invokeFunction(packageName, functionName, args) {
        seen = { packageName, functionName, args };
        return { success: true, result: { echo: "wire", timestamp: 3 } };
      },
    },
  },
  async () => $main.Echo({ message: "payload" })
);
if (result.echo !== "wire") throw new Error("transport override result");
if (seen.packageName !== "main" || seen.functionName !== "Echo") {
  throw new Error("bad invoke target " + JSON.stringify(seen));
}
if (!seen.args || seen.args[0]?.message !== "payload") {
  throw new Error("bad invoke args " + JSON.stringify(seen));
}
console.log("ok");
`)
}

func TestGenerate_acceptance_scopeRestoresAfterThrow(t *testing.T) {
	runPhase5NodeAcceptance(t, "scope-restore.mjs", `
import { withForstTestScope } from "./$testing.js";
import {
  configureDefaultInvokeClient,
  resetDefaultInvokeClientForTest,
} from "./transport/runtime.js";
import { $main } from "./pkg/main.js";

resetDefaultInvokeClientForTest();
let fetchCalls = 0;
configureDefaultInvokeClient({
  baseUrl: "http://127.0.0.1:9",
  fetchFn: async () => {
    fetchCalls += 1;
    return new Response(
      JSON.stringify({ success: true, result: { echo: "http", timestamp: 0 } }),
      { status: 200, headers: { "Content-Type": "application/json" } }
    );
  },
});

try {
  await withForstTestScope(
    {
      packages: {
        main: {
          Echo: async () => ({ echo: "scoped", timestamp: 1 }),
        },
      },
    },
    async () => {
      const r = await $main.Echo({ message: "x" });
      if (r.echo !== "scoped") throw new Error("scope failed");
      throw new Error("boom");
    }
  );
} catch (err) {
  if (err.message !== "boom") throw err;
}

const after = await $main.Echo({ message: "x" });
if (after.echo !== "http" || fetchCalls !== 1) {
  throw new Error(
    "scope leaked after throw: " + JSON.stringify({ after, fetchCalls })
  );
}
console.log("ok");
`)
}

func TestGenerate_acceptance_nestedScopesInnermostWins(t *testing.T) {
	runPhase5NodeAcceptance(t, "nested-scopes.mjs", `
import { withForstTestScope } from "./$testing.js";
import { $main } from "./pkg/main.js";

await withForstTestScope(
  {
    packages: {
      main: { Echo: async () => ({ echo: "outer", timestamp: 1 }) },
    },
  },
  async () => {
    await withForstTestScope(
      {
        packages: {
          main: { Echo: async () => ({ echo: "inner", timestamp: 2 }) },
        },
      },
      async () => {
        const inner = await $main.Echo({ message: "x" });
        if (inner.echo !== "inner") {
          throw new Error("innermost should win: " + JSON.stringify(inner));
        }
      }
    );
    const outer = await $main.Echo({ message: "x" });
    if (outer.echo !== "outer") {
      throw new Error("outer should restore: " + JSON.stringify(outer));
    }
  }
);
console.log("ok");
`)
}

func TestGenerate_acceptance_concurrentScopesDoNotLeak(t *testing.T) {
	runPhase5NodeAcceptance(t, "concurrent-scopes.mjs", `
import { withForstTestScope } from "./$testing.js";
import { $main } from "./pkg/main.js";

const delay = (ms) => new Promise((r) => setTimeout(r, ms));

await Promise.all([
  withForstTestScope(
    {
      packages: {
        main: { Echo: async () => ({ echo: "a", timestamp: 1 }) },
      },
    },
    async () => {
      await delay(30);
      const r = await $main.Echo({ message: "x" });
      if (r.echo !== "a") throw new Error("leaked into a: " + JSON.stringify(r));
    }
  ),
  withForstTestScope(
    {
      packages: {
        main: { Echo: async () => ({ echo: "b", timestamp: 2 }) },
      },
    },
    async () => {
      await delay(5);
      const r = await $main.Echo({ message: "x" });
      if (r.echo !== "b") throw new Error("leaked into b: " + JSON.stringify(r));
    }
  ),
]);
console.log("ok");
`)
}

func TestGenerate_acceptance_unhandledCallThrowsInvokeRejected(t *testing.T) {
	runPhase5NodeAcceptance(t, "unhandled.mjs", `
import { withForstTestScope, InvokeRejected } from "./$testing.js";
import { $main } from "./pkg/main.js";

let caught;
try {
  await withForstTestScope({}, async () => {
    await $main.Echo({ message: "x" });
  });
} catch (err) {
  caught = err;
}
if (!(caught instanceof InvokeRejected) && caught?._tag !== "InvokeRejected") {
  throw new Error("expected InvokeRejected, got " + caught);
}
if (caught.packageName !== "main" || caught.functionName !== "Echo") {
  throw new Error("InvokeRejected missing names: " + JSON.stringify(caught));
}
console.log("ok");
`)
}

func TestGenerate_acceptance_middlewareHooksFire(t *testing.T) {
	runPhase5NodeAcceptance(t, "middleware.mjs", `
import { createForstClient } from "./index.js";

const events = [];
const client = createForstClient({
  baseUrl: "http://127.0.0.1:9",
  fetchFn: async () =>
    new Response(
      JSON.stringify({ success: true, result: { echo: "mw", timestamp: 0 } }),
      { status: 200, headers: { "Content-Type": "application/json" } }
    ),
  middleware: [
    {
      onStart(ctx) {
        events.push(["start", ctx.packageName, ctx.functionName, ctx.attempt]);
      },
      onSuccess(ctx) {
        events.push(["success", ctx.packageName, ctx.functionName]);
      },
      onFailure() {
        events.push(["failure"]);
      },
    },
  ],
});

const result = await client.main.Echo({ message: "x" });
if (result.echo !== "mw") throw new Error("bad result");
if (events.length !== 2) throw new Error("events=" + JSON.stringify(events));
if (events[0][0] !== "start" || events[0][1] !== "main" || events[0][2] !== "Echo") {
  throw new Error("bad onStart " + JSON.stringify(events[0]));
}
if (events[1][0] !== "success") throw new Error("bad onSuccess");
console.log("ok");
`)
}

func TestGenerate_acceptance_testdataExampleRunsWithoutHTTP(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not found")
	}
	dir := t.TempDir()
	prepareMinimalGenerateProject(t, dir)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	dist := defaultClientDistDir(dir)
	exampleSrc, err := os.ReadFile(filepath.Join("testdata", "testing", "scoped-override.mjs"))
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dist, "testdata-example.mjs")
	if err := os.WriteFile(script, exampleSrc, 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("node", script)
	cmd.Dir = dist
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("testdata example failed: %v\n%s", err, out)
	}
}

func runPhase5NodeAcceptance(t *testing.T, name, body string) {
	t.Helper()
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not found")
	}
	dir := t.TempDir()
	prepareMinimalGenerateProject(t, dir)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	dist := defaultClientDistDir(dir)
	script := filepath.Join(dist, name)
	if err := os.WriteFile(script, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("node", script)
	cmd.Dir = dist
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s failed: %v\n%s", name, err, out)
	}
}
