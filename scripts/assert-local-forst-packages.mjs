#!/usr/bin/env node
/**
 * Fail if @forst/* deps are not file: paths into repoRoot/packages (no registry semver).
 * Usage: node assert-local-forst-packages.mjs <projectDir> <repoRoot>
 */
import { createRequire } from "node:module";
import { existsSync, readFileSync, realpathSync } from "node:fs";
import { join, resolve } from "node:path";

const projectDir = resolve(process.argv[2] ?? "");
const repoRoot = resolve(process.argv[3] ?? "");
if (!projectDir || !repoRoot) {
  console.error("usage: assert-local-forst-packages.mjs <projectDir> <repoRoot>");
  process.exit(1);
}

const packagesDir = resolve(repoRoot, "packages");
const pkgPath = join(projectDir, "package.json");
const pkg = JSON.parse(readFileSync(pkgPath, "utf8"));

const names = ["cli", "client", "node-runtime", "sidecar"];
const specs = new Map();

for (const field of ["dependencies", "devDependencies", "overrides"]) {
  const block = pkg[field];
  if (!block || typeof block !== "object") continue;
  for (const name of names) {
    const key = `@forst/${name}`;
    if (key in block) specs.set(key, block[key]);
  }
}

function expectedPackageDir(name) {
  return resolve(packagesDir, name);
}

function assertFileSpec(name, spec) {
  const label = `@forst/${name}`;
  if (typeof spec !== "string" || !spec.startsWith("file:")) {
    throw new Error(`${label} must use file: (got ${JSON.stringify(spec)})`);
  }
  const target = resolve(spec.slice("file:".length));
  const want = expectedPackageDir(name);
  if (!existsSync(target)) {
    throw new Error(`${label} file: target missing: ${target}`);
  }
  const got = realpathSync(target);
  const expected = realpathSync(want);
  if (got !== expected) {
    throw new Error(`${label} file: points to ${got}, expected ${expected}`);
  }
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
    assertFileSpec(name, spec);
    console.log(`ok: ${label} -> file:${expectedPackageDir(name)}`);
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
