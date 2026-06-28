import assert from "node:assert";
import test from "node:test";
import { fileURLToPath } from "node:url";
import path from "node:path";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const {
  shouldFallbackToPathOnCliError,
  handleCliResolutionFailure,
  compilerResolutionCacheKey,
  clearCompilerResolutionCache,
} = await import(path.join(__dirname, "..", "out", "compilerResolution.js"));

test("shouldFallbackToPathOnCliError is false when downloads are enabled", () => {
  assert.strictEqual(shouldFallbackToPathOnCliError(true), false);
  assert.strictEqual(shouldFallbackToPathOnCliError(false), true);
});

test("handleCliResolutionFailure rethrows when downloadCompiler is true", () => {
  const err = new Error("digest missing");
  assert.throws(
    () => handleCliResolutionFailure(err, "forst", true),
    (e) => e === err
  );
});

test("handleCliResolutionFailure returns PATH fallback when downloads are disabled", () => {
  assert.strictEqual(
    handleCliResolutionFailure(new Error("missing"), "forst", false),
    "forst"
  );
});

test("compilerResolutionCacheKey changes when download settings change", () => {
  const base = {
    forstPath: "forst",
    port: 8081,
    logLevel: "info",
    autoStart: true,
    downloadCompiler: true,
    preferLatestCompilerRelease: true,
  };
  const k1 = compilerResolutionCacheKey(base);
  const k2 = compilerResolutionCacheKey({
    ...base,
    downloadCompiler: false,
  });
  assert.notStrictEqual(k1, k2);
});

test("clearCompilerResolutionCache is callable", () => {
  clearCompilerResolutionCache();
});
