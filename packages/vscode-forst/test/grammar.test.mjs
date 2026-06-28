import assert from "node:assert";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const grammarPath = path.join(
  __dirname,
  "..",
  "syntaxes",
  "forst.tmLanguage.json"
);

test("forst.tmLanguage.json highlights use and with keywords", () => {
  const grammar = JSON.parse(fs.readFileSync(grammarPath, "utf8"));
  const control = grammar.repository.keywords.patterns.find(
    (p) => p.name === "keyword.control.forst"
  );
  assert.ok(control?.match, "keyword.control.forst pattern");
  assert.match(control.match, /\buse\b/);
  assert.match(control.match, /\bwith\b/);
});
