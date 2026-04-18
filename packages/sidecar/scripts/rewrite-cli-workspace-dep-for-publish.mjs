/**
 * Release helper: set `dependencies["@forst/cli"]` to `^<cli semver>` from `packages/cli/package.json`
 * for published tarballs (never ship `workspace:*` to npm/JSR).
 * Runs for `workspace:*` or any caret that is out of sync with the CLI package version on the tag.
 *
 * Must run last before npm/jsr pack: a root `bun install` after this can re-normalize workspace
 * members back to `workspace:*`, so `prepublish:sidecar` ends with this script + assert-published-cli-dep.
 *
 * VSIX packaging (`task build:vsix`) uses the same script for `packages/vscode-forst`.
 * Committed trees keep `workspace:*` for local dev; run `git checkout -- packages/sidecar/package.json` after a dry-run publish if needed.
 *
 * Usage:
 *   node ./scripts/rewrite-cli-workspace-dep-for-publish.mjs [path/to/package.json]
 * Default path: this package's package.json (packages/sidecar).
 */
import { readFileSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const dir = dirname(fileURLToPath(import.meta.url));
const repoRoot = join(dir, "..", "..", "..");
const defaultPkgPath = join(dir, "..", "package.json");
const cliPkgPath = join(repoRoot, "packages", "cli", "package.json");

const pkgPath = process.argv[2]
  ? resolve(process.cwd(), process.argv[2])
  : defaultPkgPath;

const pkg = JSON.parse(readFileSync(pkgPath, "utf8"));
pkg.dependencies = pkg.dependencies ?? {};

const cliPkg = JSON.parse(readFileSync(cliPkgPath, "utf8"));
const v = cliPkg.version;
if (!v || typeof v !== "string") {
  throw new Error(
    `rewrite-cli-workspace-dep-for-publish: missing packages/cli version (read ${cliPkgPath})`
  );
}

const desired = `^${v}`;
const current = pkg.dependencies["@forst/cli"];

if (current === desired) {
  console.log(
    `rewrite-cli-workspace-dep-for-publish: ${pkgPath} already has @forst/cli ${desired}`
  );
  process.exit(0);
}

pkg.dependencies["@forst/cli"] = desired;
writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + "\n");
console.log(
  `rewrite-cli-workspace-dep-for-publish: ${pkgPath} → dependencies["@forst/cli"] = ${desired} (was ${JSON.stringify(current)})`
);
