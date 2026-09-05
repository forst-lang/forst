#!/usr/bin/env node
/**
 * Fail if @forst/* deps are not local file: packed tarballs (or package dirs).
 * Usage: node assert-local-forst-packages.mjs <projectDir> [packDir]
 */
import { createRequire } from "node:module";
import { existsSync, readFileSync, realpathSync, statSync } from "node:fs";
import { basename, join, resolve } from "node:path";

const projectDir = resolve(process.argv[2] ?? "");
const packDir = process.argv[3] ? resolve(process.argv[3]) : "";
if (!projectDir) {
  console.error("usage: assert-local-forst-packages.mjs <projectDir> [packDir]");
  process.exit(1);
}

const pkgPath = join(projectDir, "package.json");
const pkg = JSON.parse(readFileSync(pkgPath, "utf8"));

const names = ["cli", "client", "errors", "runtime", "sidecar"];
const specs = new Map();

for (const field of ["dependencies", "devDependencies", "overrides"]) {
  const block = pkg[field];
  if (!block || typeof block !== "object") continue;
  for (const name of names) {
    const key = `@forst/${name}`;
    if (key in block) specs.set(key, block[key]);
  }
}

function assertFileSpec(name, spec) {
  const label = `@forst/${name}`;
  if (typeof spec !== "string" || !spec.startsWith("file:")) {
    throw new Error(`${label} must use file: (got ${JSON.stringify(spec)})`);
  }
  const target = resolve(spec.slice("file:".length));
  if (!existsSync(target)) {
    throw new Error(`${label} file: target missing: ${target}`);
  }
  const st = statSync(target);
  if (st.isFile()) {
    const base = basename(target);
    const wantPrefix = `forst-${name}-`;
    if (!base.startsWith(wantPrefix) || !base.endsWith(".tgz")) {
      throw new Error(`${label} file: tarball must be named ${wantPrefix}*.tgz (got ${base})`);
    }
    if (packDir) {
      const packRoot = realpathSync(packDir);
      const gotRoot = realpathSync(join(target, ".."));
      if (gotRoot !== packRoot) {
        throw new Error(`${label} tarball not under packDir ${packRoot} (got ${gotRoot})`);
      }
    }
    return `file:${target}`;
  }
  if (st.isDirectory()) {
    // Allow package-dir file: for manual workflows; CI uses packed tarballs.
    return `file:${realpathSync(target)}`;
  }
  throw new Error(`${label} file: target is neither a .tgz nor a directory: ${target}`);
}

let failed = false;
for (const name of names) {
  const label = `@forst/${name}`;
  const spec = specs.get(label);
  if (!spec) {
    console.error(`missing ${label} in dependencies/overrides`);
    failed = true;
    continue;
  }
  try {
    const shown = assertFileSpec(name, spec);
    console.log(`ok: ${label} -> ${shown}`);
  } catch (err) {
    console.error(String(err));
    failed = true;
  }
}

const req = createRequire(join(projectDir, "package.json"));
for (const name of names) {
  const label = `@forst/${name}`;
  const installed = join(projectDir, "node_modules", "@forst", name);
  try {
    req.resolve(label);
    if (!existsSync(installed)) {
      throw new Error(`missing ${installed}`);
    }
    console.log(`ok: ${label} resolvable from ${projectDir}`);
  } catch (err) {
    console.error(`${label} not resolvable: ${err.message}`);
    failed = true;
  }
}

if (failed) {
  process.exit(1);
}
