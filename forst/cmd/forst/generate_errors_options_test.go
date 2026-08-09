package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	transformerts "forst/internal/transformer/ts"
)

func generatePhase4Project(t *testing.T) (projectRoot, distDir string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"generate":{"link":"never"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	writeMainFt(t, dir, generateTestMinimalValidForst)
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
	errorsJS := readDistFile(t, dist, "errors.js")
	for _, name := range transformerts.ErrorClassNames() {
		if !strings.Contains(errorsJS, "export class "+name) {
			t.Fatalf("errors.js missing class %s:\n%s", name, errorsJS)
		}
	}
	assertNoEffectImport(t, errorsJS)
	assertNoEffectImport(t, readDistFile(t, dist, "transport.js"))
}

func TestGenerate_errorClassNamesHaveNoErrorSuffix(t *testing.T) {
	_, dist := generatePhase4Project(t)
	errorsJS := readDistFile(t, dist, "errors.js")
	re := regexp.MustCompile(`Error$`)
	classRe := regexp.MustCompile(`export class (\w+)`)
	for _, m := range classRe.FindAllStringSubmatch(errorsJS, -1) {
		if re.MatchString(m[1]) {
			t.Fatalf("emitted class %q ends in Error", m[1])
		}
	}
	for _, name := range transformerts.ErrorClassNames() {
		if re.MatchString(name) {
			t.Fatalf("catalog class %q ends in Error", name)
		}
	}
}

func TestGenerate_taggedErrorCarriesLiteralTag(t *testing.T) {
	_, dist := generatePhase4Project(t)
	errorsJS := readDistFile(t, dist, "errors.js")
	for _, frag := range []string{
		`Object.defineProperty(this, "_tag"`,
		"enumerable: true",
		"writable: false",
		"value: tag",
	} {
		if !strings.Contains(errorsJS, frag) {
			t.Fatalf("missing _tag contract fragment %q", frag)
		}
	}
}

func TestGenerate_taggedErrorExtendsErrorAndKeepsInstanceof(t *testing.T) {
	_, dist := generatePhase4Project(t)
	errorsJS := readDistFile(t, dist, "errors.js")
	for _, frag := range []string{
		"class extends Error",
		"Object.setPrototypeOf(this, new.target.prototype)",
		"this.name = tag",
	} {
		if !strings.Contains(errorsJS, frag) {
			t.Fatalf("missing instanceof contract fragment %q", frag)
		}
	}
	errorsDTS := readDistFile(t, dist, "errors.d.ts")
	for _, name := range transformerts.ErrorClassNames() {
		if !strings.Contains(errorsDTS, "export declare class "+name+" extends Error") {
			t.Fatalf("%s must extend Error in DTS", name)
		}
	}
}

func TestGenerate_taggedErrorAssignsPropsAsOwnProperties(t *testing.T) {
	_, dist := generatePhase4Project(t)
	errorsJS := readDistFile(t, dist, "errors.js")
	if !strings.Contains(errorsJS, "Object.assign(this, props)") {
		t.Fatal("tagged ctor must assign props as own properties")
	}
}

func TestGenerate_taggedErrorPropsCannotOverwriteTag(t *testing.T) {
	_, dist := generatePhase4Project(t)
	errorsJS := readDistFile(t, dist, "errors.js")
	assignIdx := strings.Index(errorsJS, "Object.assign(this, props)")
	defineIdx := strings.Index(errorsJS, `Object.defineProperty(this, "_tag"`)
	if assignIdx < 0 || defineIdx < 0 || assignIdx > defineIdx {
		t.Fatal("Object.assign must precede defineProperty(_tag)")
	}
}

func TestGenerate_emitsInvokeFailureUnionAndGuard(t *testing.T) {
	_, dist := generatePhase4Project(t)
	errorsDTS := readDistFile(t, dist, "errors.d.ts")
	errorsJS := readDistFile(t, dist, "errors.js")
	indexJS := readDistFile(t, dist, "index.js")
	indexDTS := readDistFile(t, dist, "index.d.ts")
	if !strings.Contains(errorsDTS, "export type InvokeFailure =") {
		t.Fatalf("missing InvokeFailure union:\n%s", errorsDTS)
	}
	if !strings.Contains(errorsJS, "export const isInvokeFailure") {
		t.Fatalf("missing isInvokeFailure:\n%s", errorsJS)
	}
	for _, name := range transformerts.AllExportedErrorClassNames() {
		if !strings.Contains(indexJS, name) {
			t.Fatalf("index.js must re-export %s", name)
		}
		if !strings.Contains(indexDTS, name) {
			t.Fatalf("index.d.ts must re-export %s", name)
		}
	}
	if !strings.Contains(indexJS, "isInvokeFailure") || !strings.Contains(indexDTS, "isInvokeFailure") {
		t.Fatal("root must re-export isInvokeFailure")
	}
	if !strings.Contains(indexDTS, "InvokeFailure") {
		t.Fatal("root must re-export InvokeFailure")
	}
}

func TestGenerate_emitsInvokeStreamAborted(t *testing.T) {
	_, dist := generatePhase4Project(t)
	errorsJS := readDistFile(t, dist, "errors.js")
	errorsDTS := readDistFile(t, dist, "errors.d.ts")
	transport := readDistFile(t, dist, "transport.js")
	if !strings.Contains(errorsJS, "export class InvokeStreamAborted") {
		t.Fatal("errors.js must define InvokeStreamAborted")
	}
	if !strings.Contains(errorsDTS, `readonly _tag: "InvokeStreamAborted"`) {
		t.Fatal("errors.d.ts must declare InvokeStreamAborted tag")
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
		"error: InvokeFailure",
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
	transportDTS := readDistFile(t, dist, "transport.d.ts")
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
	if !strings.Contains(core, `import { getDefaultInvokeClient } from "../transport.js"`) {
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
import { createInvokeClient, resetDefaultInvokeClientForTest } from "./transport.js";
import {
  InvokeRejected,
  InvokeHttpFailure,
  isInvokeFailure,
} from "./errors.js";

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
if (rejected._tag !== "InvokeRejected") {
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
  if (!(err instanceof InvokeHttpFailure) || err._tag !== "InvokeHttpFailure") {
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
	transport := readDistFile(t, dist, "transport.js")
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
import { createInvokeClient, resetDefaultInvokeClientForTest } from "./transport.js";
import { InvokeBaseUrlMissing, isInvokeFailure } from "./errors.js";

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
if (!(hit instanceof InvokeBaseUrlMissing) || hit._tag !== "InvokeBaseUrlMissing") {
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

func TestGenerate_packageJSONHasNoErrorsSubpath(t *testing.T) {
	dir, _ := generatePhase4Project(t)
	raw, err := os.ReadFile(filepath.Join(defaultClientOutDir(dir), "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"./errors"`) {
		t.Fatal("errors must not be a public exports subpath")
	}
}

func TestGenerate_transportNeverThrowsBareError(t *testing.T) {
	_, dist := generatePhase4Project(t)
	transport := readDistFile(t, dist, "transport.js")
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
