/**
 * vsce refuses paths that escape the extension root (e.g. extension/../../package.json).
 * In a Bun workspace, `node_modules/@forst/cli` is often a symlink to `packages/cli`, so
 * `vsce package` follows it and fails. Replace each such symlink with a recursive copy before packaging.
 *
 * Bun may hoist `@forst/cli` to the monorepo root (`../../node_modules`), not under `packages/vscode-forst/node_modules`.
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const vscodeRoot = path.join(__dirname, "..");
const repoRoot = path.join(vscodeRoot, "..", "..");

const candidates = [
  path.join(vscodeRoot, "node_modules", "@forst", "cli"),
  path.join(repoRoot, "node_modules", "@forst", "cli"),
];

function materialize(cliPath) {
  if (!fs.existsSync(cliPath)) {
    return false;
  }
  let st;
  try {
    st = fs.lstatSync(cliPath);
  } catch {
    return false;
  }
  if (!st.isSymbolicLink()) {
    console.log(
      `materialize-cli-dep-for-vsix: not a symlink, skip: ${cliPath}`
    );
    return true;
  }
  const real = fs.realpathSync(cliPath);
  console.log(`materialize-cli-dep-for-vsix: copy ${real} → ${cliPath}`);
  fs.rmSync(cliPath, { force: true });
  fs.cpSync(real, cliPath, { recursive: true });
  return true;
}

let any = false;
for (const p of candidates) {
  if (fs.existsSync(p)) {
    any = true;
    materialize(p);
  }
}

if (!any) {
  console.error(
    "materialize-cli-dep-for-vsix: no node_modules/@forst/cli found (tried hoisted + local)"
  );
  process.exit(1);
}

console.log("materialize-cli-dep-for-vsix: done");
