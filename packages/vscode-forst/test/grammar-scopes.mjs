import fs from "node:fs";
import path from "node:path";
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";
import textmate from "vscode-textmate";
import oniguruma from "vscode-oniguruma";

const { Registry } = textmate;
const { loadWASM, OnigScanner, OnigString } = oniguruma;

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const require = createRequire(import.meta.url);
const grammarPath = path.join(__dirname, "..", "syntaxes", "forst.tmLanguage.json");
const wasmPath = path.join(
  path.dirname(require.resolve("vscode-oniguruma/package.json")),
  "release",
  "onig.wasm"
);

let registryPromise;

async function getRegistry() {
  if (!registryPromise) {
    registryPromise = (async () => {
      const wasmBin = fs.readFileSync(wasmPath);
      await loadWASM(wasmBin.buffer);
      const registry = new Registry({
        onigLib: Promise.resolve({
          createOnigScanner: (patterns) => new OnigScanner(patterns),
          createOnigString: (s) => new OnigString(s),
        }),
        loadGrammar: async (scopeName) => {
          if (scopeName === "source.forst") {
            return JSON.parse(fs.readFileSync(grammarPath, "utf8"));
          }
          return null;
        },
      });
      await registry.loadGrammar("source.forst");
      return registry;
    })();
  }
  return registryPromise;
}

/**
 * @param {string} line
 * @returns {Promise<Array<{ startIndex: number, endIndex: number, scopes: string[] }>>}
 */
export async function tokenizeLine(line) {
  const registry = await getRegistry();
  const grammar = await registry.loadGrammar("source.forst");
  const { tokens } = grammar.tokenizeLine(line, null);
  return tokens.map((t) => ({
    startIndex: t.startIndex,
    endIndex: t.endIndex,
    scopes: t.scopes,
  }));
}

/**
 * @param {string} line
 * @param {number} index
 * @returns {Promise<string[]>}
 */
export async function scopesAt(line, index) {
  const tokens = await tokenizeLine(line);
  const token = tokens.find((t) => index >= t.startIndex && index < t.endIndex);
  return token?.scopes ?? ["source.forst"];
}

/**
 * @param {string} line
 * @param {string} needle
 * @param {string} expectedScopeSuffix
 */
export async function assertScopeOn(line, needle, expectedScopeSuffix) {
  const start = line.indexOf(needle);
  if (start < 0) {
    throw new Error(`needle ${JSON.stringify(needle)} not found in ${JSON.stringify(line)}`);
  }
  const mid = start + Math.floor(needle.length / 2);
  const scopes = await scopesAt(line, mid);
  const hit = scopes.find((s) => s === expectedScopeSuffix || s.endsWith(`.${expectedScopeSuffix}`));
  if (!hit) {
    const all = await tokenizeLine(line);
    throw new Error(
      `expected scope containing ${expectedScopeSuffix} on ${JSON.stringify(needle)} in ${JSON.stringify(line)}; got ${JSON.stringify(scopes)}; tokens=${JSON.stringify(all)}`
    );
  }
}
