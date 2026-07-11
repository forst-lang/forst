import assert from "node:assert";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";
import { assertScopeOn } from "./grammar-scopes.mjs";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const grammarPath = path.join(__dirname, "..", "syntaxes", "forst.tmLanguage.json");
const docsGrammarPath = path.join(__dirname, "..", "..", "..", "docs", "languages", "forst.json");

test("forst.tmLanguage.json highlights use and with keywords", () => {
  const grammar = JSON.parse(fs.readFileSync(grammarPath, "utf8"));
  const control = grammar.repository.keywords.patterns.find(
    (p) => p.name === "keyword.control"
  );
  assert.ok(control?.match, "keyword.control pattern");
  assert.match(control.match, /\buse\b/);
  assert.match(control.match, /\bwith\b/);
});

test("docs/languages/forst.json stays in sync with extension grammar", () => {
  const source = JSON.parse(fs.readFileSync(grammarPath, "utf8"));
  const docs = JSON.parse(fs.readFileSync(docsGrammarPath, "utf8"));
  assert.deepStrictEqual(docs.patterns, source.patterns);
  assert.deepStrictEqual(docs.repository, source.repository);
  assert.deepStrictEqual(docs.scopeName, source.scopeName);
  assert.deepStrictEqual(docs.fileTypes, ["ft"]);
  assert.equal(docs.name, "forst");
});

test("hover field type: result.id: String", async () => {
  const line = "result.id: String";
  await assertScopeOn(line, "result", "variable.other.readwrite");
  await assertScopeOn(line, "id", "variable.other.property");
  await assertScopeOn(line, "String", "support.type.primitive");
});

test("hover signature: checkout(amount Float, currency String) -> String", async () => {
  const line = "checkout(amount Float, currency String) -> String";
  await assertScopeOn(line, "checkout", "entity.name.function");
  await assertScopeOn(line, "amount", "variable.parameter");
  await assertScopeOn(line, "Float", "support.type.primitive");
  await assertScopeOn(line, "currency", "variable.parameter");
  await assertScopeOn(line, "String", "support.type.primitive");
});

test("constraint chain: String.Min(1).Max(10)", async () => {
  const line = "String.Min(1).Max(10)";
  await assertScopeOn(line, "String", "support.type.primitive");
  await assertScopeOn(line, "Min", "entity.name.function.constraint.forst");
  await assertScopeOn(line, "Max", "entity.name.function.constraint.forst");
});
