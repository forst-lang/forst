// Minimal persistent host shim for multi-package-dev hostMode reload e2e.
import { pathToFileURL } from "node:url";
import path from "node:path";

const repoRoot = process.env.FORST_REPO_ROOT;
if (!repoRoot) {
  console.error("FORST_REPO_ROOT is required");
  process.exit(1);
}

const hostJS = path.join(repoRoot, "packages", "node-runtime", "dist", "host.js");
const { signalForstAppReady } = await import(pathToFileURL(hostJS).href);
await signalForstAppReady();
await new Promise(() => {});
