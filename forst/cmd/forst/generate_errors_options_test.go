package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	transformerts "forst/internal/transformer/ts"
)

func generatePhase4Project(t *testing.T) (projectRoot, distDir string) {
	t.Helper()
	dir := t.TempDir()
	prepareMinimalGenerateProject(t, dir)
	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}
	return dir, defaultClientDistDir(dir)
}

func readDistFile(t *testing.T, distDir, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(distDir, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(raw)
}

func TestGenerate_emitsTaggedErrorClasses(t *testing.T) {
	_, dist := generatePhase4Project(t)
	errorsJS := readDistFile(t, dist, transformerts.InfraErrorsSubpath+".js")
	if !strings.Contains(errorsJS, "export {};") {
		t.Fatalf("$errors.js should be a domain-only stub:\n%s", errorsJS)
	}
	for _, name := range transformerts.ErrorClassNames() {
		if strings.Contains(errorsJS, name) {
			t.Fatalf("$errors.js must not re-export invoke error %s:\n%s", name, errorsJS)
		}
	}
	transport := readDistFile(t, dist, "transport/runtime.js")
	for _, name := range transformerts.ErrorClassNames() {
		if !strings.Contains(transport, name) {
			t.Fatalf("transport/runtime.js missing invoke error %s:\n%s", name, transport)
		}
	}
	if !strings.Contains(transport, `from "@forst/errors"`) {
		t.Fatalf("transport must import invoke errors from @forst/errors:\n%s", transport)
	}
	if _, err := os.Stat(filepath.Join(dist, "invoke-errors.js")); !os.IsNotExist(err) {
		t.Fatal("invoke-errors.js must not be generated")
	}
	assertNoEffectImport(t, errorsJS)
	assertNoEffectImport(t, transport)
}

func TestGenerate_errorClassNamesHaveNoErrorSuffix(t *testing.T) {
	for _, name := range transformerts.ErrorClassNames() {
		if strings.HasSuffix(name, "Error") {
			t.Fatalf("catalog class %q ends in Error", name)
		}
	}
}

func TestGenerate_errorsModuleIsDomainOnlyStub(t *testing.T) {
	_, dist := generatePhase4Project(t)
	errorsJS := readDistFile(t, dist, transformerts.InfraErrorsSubpath+".js")
	errorsDTS := readDistFile(t, dist, transformerts.InfraErrorsSubpath+".d.ts")
	if !strings.Contains(errorsJS, "export {};") {
		t.Fatalf("$errors.js missing export stub:\n%s", errorsJS)
	}
	if !strings.Contains(errorsDTS, "export {};") {
		t.Fatalf("$errors.d.ts missing export stub:\n%s", errorsDTS)
	}
	assertContainsNoneGenerate(t, errorsJS, []string{
		"isInvokeFailure",
		"export class InvokeRejected",
		"const tagged =",
		`from "@forst/errors";`,
	})
}

func assertContainsNoneGenerate(t *testing.T, got string, frags []string) {
	t.Helper()
	for _, frag := range frags {
		if strings.Contains(got, frag) {
			t.Fatalf("unexpected fragment %q in:\n%s", frag, got)
		}
	}
}

func TestGenerate_emitsInvokeFailureUnionAndGuard(t *testing.T) {
	dir, dist := generatePhase4Project(t)
	errorsJS := readDistFile(t, dist, transformerts.InfraErrorsSubpath+".js")
	indexJS := readDistFile(t, dist, "index.js")
	indexDTS := readDistFile(t, dist, "index.d.ts")
	coreDTS := readDistFile(t, dist, "core/main.d.ts")
	if !strings.Contains(coreDTS, `from "@forst/errors"`) {
		t.Fatalf("core/main.d.ts should import invoke failure types from @forst/errors:\n%s", coreDTS)
	}
	if !strings.Contains(coreDTS, "InvokeFailure") {
		t.Fatalf("core/main.d.ts missing InvokeFailure:\n%s", coreDTS)
	}
	if strings.Contains(errorsJS, "isInvokeFailure") {
		t.Fatalf("$errors.js must not re-export isInvokeFailure:\n%s", errorsJS)
	}
	for _, name := range []string{"ForstUnknownFailure", "BcryptGenerateFailed"} {
		if strings.Contains(indexJS, name) {
			t.Fatalf("index.js must not re-export error %s", name)
		}
		if strings.Contains(indexDTS, name) {
			t.Fatalf("index.d.ts must not re-export error %s", name)
		}
	}
	for _, name := range transformerts.ErrorClassNames() {
		if strings.Contains(indexJS, name) {
			t.Fatalf("index.js must not re-export invoke error %s", name)
		}
	}
	if strings.Contains(indexJS, "isInvokeFailure") || strings.Contains(indexDTS, "isInvokeFailure") {
		t.Fatal("root must not re-export isInvokeFailure")
	}
	if strings.Contains(indexDTS, "InvokeFailure") {
		t.Fatal("root must not re-export InvokeFailure")
	}
	raw, err := os.ReadFile(filepath.Join(defaultClientOutDir(dir), "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	pkgJSON := string(raw)
	for _, subpath := range []string{`"./` + transformerts.InfraErrorsSubpath + `"`, transformerts.ErrorsPackageName} {
		if !strings.Contains(pkgJSON, subpath) {
			t.Fatalf("package.json must mention %s:\n%s", subpath, pkgJSON)
		}
	}
	if strings.Contains(pkgJSON, `"./invoke"`) {
		t.Fatalf("package.json must not export ./invoke:\n%s", pkgJSON)
	}
}

func TestGenerate_emitsInvokeStreamAborted(t *testing.T) {
	_, dist := generatePhase4Project(t)
	errorsJS := readDistFile(t, dist, transformerts.InfraErrorsSubpath+".js")
	transport := readDistFile(t, dist, "transport/runtime.js")
	if strings.Contains(errorsJS, "InvokeStreamAborted") {
		t.Fatalf("$errors.js must not re-export InvokeStreamAborted:\n%s", errorsJS)
	}
	if !strings.Contains(transport, "new InvokeStreamAborted") {
		t.Fatal("transport must throw InvokeStreamAborted")
	}
	if strings.Contains(transport, "export class InvokeStreamAborted") {
		t.Fatal("InvokeStreamAborted must not be redefined in transport")
	}
}

func TestGenerate_coreModuleHasSafeVariant(t *testing.T) {
	_, dist := generatePhase4Project(t)
	core := readDistFile(t, dist, "core/main.js")
	dts := readDistFile(t, dist, "core/main.d.ts")
	for _, frag := range []string{
		"Echo.safe = async (",
		"{ ok: true, value: await Echo(",
		"{ ok: false, error }",
	} {
		if !strings.Contains(core, frag) {
			t.Fatalf("core missing safe variant fragment %q:\n%s", frag, core)
		}
	}
	for _, frag := range []string{
		"export declare namespace Echo",
		"function safe(",
		"ok: false; error:",
		"InvokeFailure",
	} {
		if !strings.Contains(dts, frag) {
			t.Fatalf("core.d.ts missing safe fragment %q:\n%s", frag, dts)
		}
	}
}

func TestGenerate_functionAcceptsInvokeCallOptions(t *testing.T) {
	_, dist := generatePhase4Project(t)
	core := readDistFile(t, dist, "core/main.js")
	dts := readDistFile(t, dist, "core/main.d.ts")
	transportDTS := readDistFile(t, dist, "transport/runtime.d.ts")
	for _, frag := range []string{
		"export async function Echo(input, options)",
		`client.invokeFunction("main", "Echo", [input], options)`,
	} {
		if !strings.Contains(core, frag) {
			t.Fatalf("core missing options fragment %q", frag)
		}
	}
	if !strings.Contains(dts, "options?: InvokeCallOptions") {
		t.Fatalf("core.d.ts missing InvokeCallOptions:\n%s", dts)
	}
	for _, frag := range []string{
		"signal?: AbortSignal",
		"timeoutMs?: number",
		"retries?: number",
		"transport?: ForstInvokeClient",
	} {
		if !strings.Contains(transportDTS, frag) {
			t.Fatalf("InvokeCallOptions missing %q", frag)
		}
	}
}

func TestGenerate_coreUsesOptionsTransportWhenProvided(t *testing.T) {
	_, dist := generatePhase4Project(t)
	core := readDistFile(t, dist, "core/main.js")
	if !strings.Contains(core, "options?.transport ?? getDefaultInvokeClient()") {
		t.Fatalf("core must prefer options.transport:\n%s", core)
	}
}

func TestGenerate_coreFallsBackToDefaultClientWhenNoTransport(t *testing.T) {
	_, dist := generatePhase4Project(t)
	core := readDistFile(t, dist, "core/main.js")
	if !strings.Contains(core, `import { getDefaultInvokeClient } from "../transport/runtime.js"`) {
		t.Fatal("core must import getDefaultInvokeClient")
	}
	if !strings.Contains(core, "options?.transport ?? getDefaultInvokeClient()") {
		t.Fatal("core must fall back to getDefaultInvokeClient")
	}
}

func TestGenerate_acceptance_typedInvokeError(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not found")
	}
	_, dist := generatePhase4Project(t)
	script := filepath.Join(dist, "_phase4_typed_error.mjs")
	body := `
import { createInvokeClient, resetDefaultInvokeClientForTest } from "./transport/runtime.js";
import {
  InvokeRejected,
  InvokeHttpFailure,
  isInvokeFailure,
} from "@forst/errors";

resetDefaultInvokeClientForTest();
const client = createInvokeClient({
  baseUrl: "http://127.0.0.1:9",
  fetchFn: async () =>
    new Response(JSON.stringify({ success: false, error: "bad password" }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
});

let rejected;
try {
  await client.invokeFunction("main", "Echo", [{ message: "x" }]);
  console.error("expected throw");
  process.exit(1);
} catch (err) {
  rejected = err;
}
if (!(rejected instanceof InvokeRejected)) {
  console.error("not InvokeRejected", rejected);
  process.exit(1);
}
if (rejected._tag !== "@forst/errors/InvokeRejected") {
  console.error("bad tag", rejected._tag);
  process.exit(1);
}
if (!isInvokeFailure(rejected)) {
  console.error("isInvokeFailure failed");
  process.exit(1);
}
if (rejected.serverError !== "bad password") {
  console.error("bad serverError", rejected.serverError);
  process.exit(1);
}

resetDefaultInvokeClientForTest();
const httpClient = createInvokeClient({
  baseUrl: "http://127.0.0.1:9",
  fetchFn: async () => new Response("nope", { status: 503 }),
});
try {
  await httpClient.invokeFunction("main", "Echo", []);
  console.error("expected http throw");
  process.exit(1);
} catch (err) {
  if (!(err instanceof InvokeHttpFailure) || err._tag !== "@forst/errors/InvokeHttpFailure") {
    console.error("not InvokeHttpFailure", err);
    process.exit(1);
  }
  if (err.status !== 503) {
    console.error("bad status", err.status);
    process.exit(1);
  }
}
console.log("ok");
`
	if err := os.WriteFile(script, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("node", script)
	cmd.Dir = dist
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node typed error smoke failed: %v\n%s", err, out)
	}
}

func TestGenerate_acceptance_noSpawnInProduction(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not found")
	}
	_, dist := generatePhase4Project(t)
	transport := readDistFile(t, dist, "transport/runtime.js")
	for _, banned := range []string{"child_process", "spawn(", "fork("} {
		if strings.Contains(transport, banned) {
			t.Fatalf("transport must not spawn (%q present)", banned)
		}
	}
	if !strings.Contains(transport, "NODE_ENV") || !strings.Contains(transport, "InvokeBaseUrlMissing") {
		t.Fatal("transport must guard production with InvokeBaseUrlMissing")
	}

	script := filepath.Join(dist, "_phase4_no_spawn_prod.mjs")
	body := `
import { createInvokeClient, resetDefaultInvokeClientForTest } from "./transport/runtime.js";
import { InvokeBaseUrlMissing, isInvokeFailure } from "@forst/errors";

const prev = process.env.NODE_ENV;
const prevBase = process.env.FORST_BASE_URL;
const prevInvoke = process.env.FORST_INVOKE_URL;
const prevDev = process.env.FORST_DEV_URL;
delete process.env.FORST_BASE_URL;
delete process.env.FORST_INVOKE_URL;
delete process.env.FORST_DEV_URL;
process.env.NODE_ENV = "production";
resetDefaultInvokeClientForTest();
let hit;
try {
  createInvokeClient({ allowSpawn: true, transport: "dev" });
  console.error("expected InvokeBaseUrlMissing");
  process.exit(1);
} catch (err) {
  hit = err;
} finally {
  if (prev === undefined) delete process.env.NODE_ENV;
  else process.env.NODE_ENV = prev;
  if (prevBase === undefined) delete process.env.FORST_BASE_URL;
  else process.env.FORST_BASE_URL = prevBase;
  if (prevInvoke === undefined) delete process.env.FORST_INVOKE_URL;
  else process.env.FORST_INVOKE_URL = prevInvoke;
  if (prevDev === undefined) delete process.env.FORST_DEV_URL;
  else process.env.FORST_DEV_URL = prevDev;
}
if (!(hit instanceof InvokeBaseUrlMissing) || hit._tag !== "@forst/errors/InvokeBaseUrlMissing") {
  console.error("bad error", hit);
  process.exit(1);
}
if (!isInvokeFailure(hit) || hit.nodeEnv !== "production" || hit.envVar !== "FORST_BASE_URL") {
  console.error("bad fields", hit);
  process.exit(1);
}
console.log("ok");
`
	if err := os.WriteFile(script, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("node", script)
	cmd.Dir = dist
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node production spawn guard failed: %v\n%s", err, out)
	}
}

func TestGenerate_packageJSONExportsErrorsSubpath(t *testing.T) {
	dir, _ := generatePhase4Project(t)
	raw, err := os.ReadFile(filepath.Join(defaultClientOutDir(dir), "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if !strings.Contains(body, `"./`+transformerts.InfraErrorsSubpath+`"`) {
		t.Fatal("package.json must export ./errors subpath")
	}
}

func TestGenerate_transportNeverThrowsBareError(t *testing.T) {
	_, dist := generatePhase4Project(t)
	transport := readDistFile(t, dist, "transport/runtime.js")
	if strings.Contains(transport, "throw new Error(") {
		t.Fatal("transport must not throw bare Error for invoke failures")
	}
	assertNoEffectImport(t, transport)
}

func assertNoEffectImport(t *testing.T, src string) {
	t.Helper()
	for _, banned := range []string{
		`from "effect"`,
		`from 'effect'`,
		`require("effect")`,
		`require('effect')`,
	} {
		if strings.Contains(src, banned) {
			t.Fatalf("must not import effect (%q)", banned)
		}
	}
}
