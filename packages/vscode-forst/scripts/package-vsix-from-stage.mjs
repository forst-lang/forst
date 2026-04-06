/**
 * Run `vsce package` from a **temporary** directory that contains only this extension + a
 * materialized `node_modules/@forst/cli` (npm "files" layout). In the monorepo, `npm list` used by
 * vsce includes the **repo root** as the first line, so vsce globs the whole workspace and fails
 * with `invalid relative path: extension/../../package.json`. A standalone stage fixes that.
 */
import { spawnSync } from "node:child_process";
import fs, { mkdtempSync, rmSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

import {
  copyNpmFilesLayout,
  getPackagesCliRoot,
} from "./materialize-cli-dep-for-vsix.mjs";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const vscodeRoot = path.join(__dirname, "..");
const repoRoot = path.join(vscodeRoot, "..", "..");
/** Repo-installed CLI (`bunx vsce` can mis-parse flags under Bun). */
const vsceEntry = path.join(
  repoRoot,
  "node_modules",
  "@vscode",
  "vsce",
  "vsce"
);

const outVsix = process.argv[2];
if (!outVsix) {
  console.error("usage: node package-vsix-from-stage.mjs <output.vsix>");
  process.exit(1);
}

const outAbs = path.isAbsolute(outVsix)
  ? outVsix
  : path.join(process.cwd(), outVsix);
fs.mkdirSync(path.dirname(outAbs), { recursive: true });

const cliSource = getPackagesCliRoot();
const distIndex = path.join(cliSource, "dist", "index.js");
if (!fs.existsSync(distIndex)) {
  console.error(
    `package-vsix-from-stage: ${distIndex} missing — run task build:cli first`
  );
  process.exit(1);
}

const stage = mkdtempSync(path.join(os.tmpdir(), "forst-vsix-stage-"));

try {
  /** @type {readonly [string, { dir: boolean }][]} */
  const entries = [
    ["out", { dir: true }],
    ["language-configuration.json", { dir: false }],
    ["syntaxes", { dir: true }],
    ["snippets", { dir: true }],
    ["images", { dir: true }],
    ["icons", { dir: true }],
    ["README.md", { dir: false }],
    ["package.json", { dir: false }],
  ];

  for (const [name, { dir }] of entries) {
    const src = path.join(vscodeRoot, name);
    const dst = path.join(stage, name);
    if (!fs.existsSync(src)) {
      throw new Error(`package-vsix-from-stage: missing ${src}`);
    }
    if (dir) {
      fs.cpSync(src, dst, { recursive: true });
    } else {
      fs.copyFileSync(src, dst);
    }
  }

  const stagedPkgPath = path.join(stage, "package.json");
  const pkgJson = JSON.parse(fs.readFileSync(stagedPkgPath, "utf8"));
  if (pkgJson.scripts?.["vscode:prepublish"]) {
    delete pkgJson.scripts["vscode:prepublish"];
  }
  const cliFilesPattern = "node_modules/@forst/cli/**";
  if (!Array.isArray(pkgJson.files)) {
    pkgJson.files = [];
  }
  if (!pkgJson.files.includes(cliFilesPattern)) {
    pkgJson.files.push(cliFilesPattern);
  }
  fs.writeFileSync(stagedPkgPath, `${JSON.stringify(pkgJson, null, 2)}\n`);
  console.log(
    "package-vsix-from-stage: staged package.json — no vscode:prepublish; files include @forst/cli"
  );

  const cliDest = path.join(stage, "node_modules", "@forst", "cli");
  fs.mkdirSync(path.dirname(cliDest), { recursive: true });
  console.log(
    `package-vsix-from-stage: ${cliDest} ← npm "files" layout from ${cliSource}`
  );
  copyNpmFilesLayout(cliSource, cliDest);

  if (!fs.existsSync(vsceEntry)) {
    throw new Error(
      `package-vsix-from-stage: missing ${vsceEntry} — run bun install at repo root`
    );
  }
  const vsceArgs = [
    vsceEntry,
    "package",
    "--skip-license",
    "--allow-package-all-secrets",
    "--allow-package-env-file",
    "-o",
    outAbs,
  ];
  const r = spawnSync(process.execPath, vsceArgs, {
    cwd: stage,
    stdio: "inherit",
    env: process.env,
  });
  if (r.error) {
    throw r.error;
  }
  if (r.status !== 0) {
    process.exit(r.status ?? 1);
  }
} finally {
  rmSync(stage, { recursive: true, force: true });
}

console.log(`package-vsix-from-stage: wrote ${outAbs}`);
