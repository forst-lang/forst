#!/usr/bin/env node
/**
 * Rewrite @forst/* deps in a temp remix-serve copy to absolute file: paths under REPO/packages.
 * Usage: node patch-remix-serve-standalone-deps.mjs <projectDir> <repoRoot>
 */
import { readFileSync, writeFileSync } from "node:fs";
import { join, resolve } from "node:path";

const projectDir = resolve(process.argv[2] ?? "");
const repoRoot = resolve(process.argv[3] ?? "");
if (!projectDir || !repoRoot) {
  console.error("usage: patch-remix-serve-standalone-deps.mjs <projectDir> <repoRoot>");
  process.exit(1);
}

const pkgPath = join(projectDir, "package.json");
const pkg = JSON.parse(readFileSync(pkgPath, "utf8"));

const forstNames = ["cli", "client", "node-runtime", "sidecar"];
const fileDeps = Object.fromEntries(
  forstNames.map((name) => [
    `@forst/${name}`,
    `file:${join(repoRoot, "packages", name)}`,
  ])
);

pkg.dependencies = pkg.dependencies ?? {};
for (const [dep, filePath] of Object.entries(fileDeps)) {
  if (dep in pkg.dependencies || dep === "@forst/client" || dep === "@forst/cli" || dep === "@forst/node-runtime") {
    pkg.dependencies[dep] = filePath;
  }
}
// Ensure client-facing deps exist even if example drops direct sidecar.
pkg.dependencies["@forst/cli"] = fileDeps["@forst/cli"];
pkg.dependencies["@forst/client"] = fileDeps["@forst/client"];
pkg.dependencies["@forst/node-runtime"] = fileDeps["@forst/node-runtime"];
pkg.dependencies["@forst/sidecar"] = fileDeps["@forst/sidecar"];

pkg.devDependencies = pkg.devDependencies ?? {};
if (!pkg.devDependencies.tsx) {
  pkg.devDependencies.tsx = "^4.23.0";
}

pkg.overrides = {
  ...pkg.overrides,
  ...fileDeps,
};

writeFileSync(pkgPath, `${JSON.stringify(pkg, null, 2)}\n`);
console.log(`patched ${pkgPath} with local @forst/* file: deps`);
