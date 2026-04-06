import assert from "node:assert";
import test from "node:test";
import { fileURLToPath } from "node:url";
import path from "node:path";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const { formatForstDebugInfo } = await import(
  path.join(__dirname, "..", "out", "debugInfo.js")
);

test("formatForstDebugInfo includes key fields and compiler output", () => {
  const text = formatForstDebugInfo({
    extensionVersion: "0.1.0",
    vscodeVersion: "1.99.0",
    platform: "darwin",
    forstPathSetting: "forst",
    resolvedCompilerPath: "/x/bin/forst",
    compilerVersionOutput: "forst 1.0.0 (abc) 2026-01-01",
    downloadCompiler: true,
    lspPort: 8081,
    lspAutoStart: true,
    lspLogLevel: "info",
  });
  assert.ok(text.includes("0.1.0"));
  assert.ok(text.includes("8081"));
  assert.ok(text.includes("forst 1.0.0"));
  assert.ok(text.includes("/x/bin/forst"));
});
