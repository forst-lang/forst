/**
 * Release helper: replace `dependencies["@forst/cli"]: "workspace:*"` with `^<cli semver>` for published tarballs.
 * **@forst/sidecar** must never ship `workspace:*` to npm/JSR — consumers install from registries, not the monorepo.
 * Reads `version` from `packages/cli/package.json` (same repo / tag as the release; align with the CLI version published to npm).
 * CI runs this before sidecar npm/JSR publish; `packages/sidecar` prepublishOnly runs it too so manual `npm publish` cannot skip rewriting.
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
const current = pkg.dependencies["@forst/cli"];
if (current !== "workspace:*") {
  console.log(
    `rewrite-cli-workspace-dep-for-publish: skip ${pkgPath} (@forst/cli is "${current ?? "(missing)"}", not workspace:*)`
  );
  process.exit(0);
}

const cliPkg = JSON.parse(readFileSync(cliPkgPath, "utf8"));
const v = cliPkg.version;
if (!v || typeof v !== "string") {
  throw new Error(
    `rewrite-cli-workspace-dep-for-publish: missing packages/cli version (read ${cliPkgPath})`
  );
}

pkg.dependencies["@forst/cli"] = `^${v}`;
writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + "\n");
console.log(
  `rewrite-cli-workspace-dep-for-publish: ${pkgPath} → dependencies["@forst/cli"] = ^${v}`
);
