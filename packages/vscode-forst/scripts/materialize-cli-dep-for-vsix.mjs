/**
 * vsce refuses paths that escape the extension root (e.g. extension/../../package.json).
 * In a Bun workspace, `node_modules/@forst/cli` is usually a symlink to `packages/cli`. Replacing
 * it with a plain tree fixes that, but a full `cpSync` of `packages/cli` still copies **nested**
 * symlinks (e.g. under dev `node_modules`) that point back into the monorepo — vsce follows them and
 * fails again.
 *
 * This script rebuilds `node_modules/@forst/cli` using the same **files** layout as `npm publish`
 * for `packages/cli` (see that package.json `files` field): `dist/`, `README.md`, `LICENSE`, plus
 * `package.json`. No symlinks, minimal size.
 *
 * After copying, we run `npm install --omit=dev` in that folder so runtime deps (e.g. `semver`)
 * exist under `node_modules`. Otherwise `vsce`/`npm list --production` fails with ELSPROBLEMS.
 *
 * Bun may hoist `@forst/cli` to the repo root `node_modules`, not under `packages/vscode-forst/node_modules`.
 *
 * For production VSIX, `package-vsix-from-stage.mjs` copies the extension into a **temporary**
 * directory without a parent workspace so `npm list` (used by vsce) does not include the monorepo root.
 */
import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const vscodeRoot = path.join(__dirname, "..");
const repoRoot = path.join(vscodeRoot, "..", "..");

/** Resolved `packages/cli` absolute path for the repo containing this extension. */
export function getPackagesCliRoot() {
  return path.join(repoRoot, "packages", "cli");
}

/**
 * @param {string} sourceRoot — e.g. packages/cli
 * @param {string} destRoot — e.g. node_modules/@forst/cli
 */
export function copyNpmFilesLayout(sourceRoot, destRoot) {
  const pkgPath = path.join(sourceRoot, "package.json");
  if (!fs.existsSync(pkgPath)) {
    throw new Error(`materialize-cli-dep-for-vsix: missing ${pkgPath}`);
  }
  const pkg = JSON.parse(fs.readFileSync(pkgPath, "utf8"));
  const files = Array.isArray(pkg.files) ? pkg.files : ["dist"];

  fs.rmSync(destRoot, { recursive: true, force: true });
  fs.mkdirSync(destRoot, { recursive: true });
  fs.copyFileSync(pkgPath, path.join(destRoot, "package.json"));

  for (const entry of files) {
    const src = path.join(sourceRoot, entry);
    const dst = path.join(destRoot, entry);
    if (!fs.existsSync(src)) {
      console.warn(
        `materialize-cli-dep-for-vsix: skip missing "files" entry ${JSON.stringify(entry)}`
      );
      continue;
    }
    const st = fs.statSync(src);
    if (st.isDirectory()) {
      fs.cpSync(src, dst, { recursive: true, dereference: true });
    } else {
      fs.copyFileSync(src, dst);
    }
  }

  const npm = process.platform === "win32" ? "npm.cmd" : "npm";
  const install = spawnSync(
    npm,
    ["install", "--omit=dev", "--no-audit", "--no-fund"],
    {
      cwd: destRoot,
      stdio: "inherit",
      env: process.env,
    }
  );
  if (install.error) {
    throw install.error;
  }
  if (install.status !== 0) {
    throw new Error(
      `materialize-cli-dep-for-vsix: npm install --omit=dev failed in ${destRoot} (status ${install.status})`
    );
  }
}

const candidates = [
  path.join(vscodeRoot, "node_modules", "@forst", "cli"),
  path.join(repoRoot, "node_modules", "@forst", "cli"),
];

function isRunAsMain() {
  const mainPath = process.argv[1];
  if (!mainPath) {
    return false;
  }
  return path.resolve(mainPath) === path.resolve(fileURLToPath(import.meta.url));
}

if (isRunAsMain()) {
  const cliSource = getPackagesCliRoot();
  const distIndex = path.join(cliSource, "dist", "index.js");
  if (!fs.existsSync(distIndex)) {
    console.error(
      `materialize-cli-dep-for-vsix: ${distIndex} missing — run task build:cli (or bun run build in packages/cli) first`
    );
    process.exit(1);
  }

  for (const cliPath of candidates) {
    fs.mkdirSync(path.dirname(cliPath), { recursive: true });
    console.log(
      `materialize-cli-dep-for-vsix: ${cliPath} ← npm "files" layout from ${cliSource}`
    );
    copyNpmFilesLayout(cliSource, cliPath);
  }

  console.log("materialize-cli-dep-for-vsix: done");
}
