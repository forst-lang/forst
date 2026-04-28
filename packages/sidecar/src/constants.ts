import { readFileSync } from "node:fs";
import { join } from "node:path";

/** `dist/*.js` → `packages/sidecar/package.json` (same layout as `@forst/cli` version helper). */
const pkgPath = join(__dirname, "..", "package.json");
const pkg = JSON.parse(readFileSync(pkgPath, "utf8")) as { version: string };

/** npm semver of this `@forst/sidecar` build (from package.json next to dist/). */
export const SIDECAR_PACKAGE_VERSION = pkg.version;

/** Sent on every request to `forst dev` so server logs can attribute traffic to a sidecar release. */
export const SIDECAR_VERSION_HTTP_HEADER = "X-Forst-Sidecar-Version";

/**
 * Expected `contractVersion` from `GET /version` for this `@forst/sidecar` release.
 * Must stay in sync with `HTTPContractVersion` in `forst/internal/devserver/server.go`.
 */
export const FORST_DEV_HTTP_CONTRACT_VERSION = "2";
