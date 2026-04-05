import assert from "node:assert";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const snippetsPath = path.join(__dirname, "..", "snippets", "forst.json");

test("snippets/forst.json parses and defines expected prefixes", () => {
  const raw = fs.readFileSync(snippetsPath, "utf8");
  const data = JSON.parse(raw);
  assert.ok(data && typeof data === "object");

  const required = [
    "Package with main",
    "Function",
    "Function (no result)",
    "Type alias",
    "Import",
    "Ensure",
  ];
  for (const name of required) {
    assert.ok(name in data, `missing snippet: ${name}`);
    const s = data[name];
    assert.ok(typeof s.prefix === "string" && s.prefix.length > 0, `${name}: prefix`);
    assert.ok(Array.isArray(s.body) || typeof s.body === "string", `${name}: body`);
  }
});
