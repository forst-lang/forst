#!/usr/bin/env node
/**
 * Rewrite @forst/* deps in a temp remix-serve copy to file: paths for local packed
 * tarballs (avoids linking live monorepo package dirs, which pulls workspace
 * members and their devDependencies and can hang bun install).
 *
 * Usage: node patch-remix-serve-standalone-deps.mjs <projectDir> <packDir>
 * packDir must contain forst-<name>-*.tgz for cli, client, errors, node-runtime, sidecar.
 */
import { readdirSync, readFileSync, writeFileSync } from "node:fs";
import { join, resolve } from "node:path";

const projectDir = resolve(process.argv[2] ?? "");
const packDir = resolve(process.argv[3] ?? "");
if (!projectDir || !packDir) {
  console.error("usage: patch-remix-serve-standalone-deps.mjs <projectDir> <packDir>");
  process.exit(1);
}

const forstNames = ["cli", "client", "errors", "node-runtime", "sidecar"];
const tarballs = readdirSync(packDir).filter((f) => f.endsWith(".tgz"));

function tarballFor(name) {
  const prefix = `forst-${name}-`;
  const matches = tarballs.filter((f) => f.startsWith(prefix));
  if (matches.length === 0) {
    throw new Error(`missing packed tarball for @forst/${name} in ${packDir} (expected ${prefix}*.tgz)`);
  }
  if (matches.length > 1) {
    throw new Error(`ambiguous packed tarballs for @forst/${name}: ${matches.join(", ")}`);
  }
  return join(packDir, matches[0]);
}

const fileDeps = Object.fromEntries(
  forstNames.map((name) => [`@forst/${name}`, `file:${tarballFor(name)}`])
);

const pkgPath = join(projectDir, "package.json");
const pkg = JSON.parse(readFileSync(pkgPath, "utf8"));

pkg.dependencies = pkg.dependencies ?? {};
for (const [dep, filePath] of Object.entries(fileDeps)) {
  pkg.dependencies[dep] = filePath;
}

pkg.devDependencies = pkg.devDependencies ?? {};
if (!pkg.devDependencies.tsx) {
  pkg.devDependencies.tsx = "^4.23.0";
}

pkg.overrides = {
  ...pkg.overrides,
  ...fileDeps,
};

writeFileSync(pkgPath, `${JSON.stringify(pkg, null, 2)}\n`);
console.log(`patched ${pkgPath} with local @forst/* packed tarball deps from ${packDir}`);
