#!/usr/bin/env node
/**
 * Sync forst.tmLanguage.json to docs/languages/forst.json (Mintlify Shiki grammar).
 * Source of truth: packages/vscode-forst/syntaxes/forst.tmLanguage.json
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "../../..");
const source = path.join(
  repoRoot,
  "packages/vscode-forst/syntaxes/forst.tmLanguage.json"
);
const dest = path.join(repoRoot, "docs/languages/forst.json");

const grammar = JSON.parse(fs.readFileSync(source, "utf8"));
const docsGrammar = {
  ...grammar,
  name: "forst",
  fileTypes: ["ft"],
};
fs.writeFileSync(dest, JSON.stringify(docsGrammar, null, 2) + "\n");
console.log(`synced ${path.relative(repoRoot, source)} -> ${path.relative(repoRoot, dest)}`);
