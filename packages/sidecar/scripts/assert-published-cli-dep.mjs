/**
 * Fail if `dependencies["@forst/cli"]` is still a workspace protocol or otherwise
 * unsuitable for a published registry package. Run immediately after
 * rewrite-cli-workspace-dep-for-publish.mjs (nothing after that should rewrite package.json).
 *
 * This script only reads `package.json` on disk — it does **not** call the npm registry.
 * (Despite the filename: “published” means “fit for npm/JSR tarballs”, not “already on npm”.)
 */
import { readFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const dir = dirname(fileURLToPath(import.meta.url));
const defaultPkgPath = join(dir, "..", "package.json");

const pkgPath = process.argv[2]
  ? resolve(process.cwd(), process.argv[2])
  : defaultPkgPath;

const pkg = JSON.parse(readFileSync(pkgPath, "utf8"));
const dep = pkg.dependencies?.["@forst/cli"];

if (dep == null || typeof dep !== "string") {
  console.error(
    `assert-published-cli-dep: missing dependencies["@forst/cli"] in ${pkgPath}`
  );
  process.exit(1);
}

if (dep.startsWith("workspace:")) {
  console.error(
    `assert-published-cli-dep: dependencies["@forst/cli"] must not use workspace protocol (got ${JSON.stringify(dep)})`
  );
  process.exit(1);
}

if (!dep.startsWith("^")) {
  console.error(
    `assert-published-cli-dep: expected caret range from rewrite script (got ${JSON.stringify(dep)})`
  );
  process.exit(1);
}

console.log(`assert-published-cli-dep: ok — @forst/cli ${dep}`);
