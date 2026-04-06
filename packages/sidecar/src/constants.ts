import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const pkg = require("../package.json") as { version: string };

/** npm semver of this `@forst/sidecar` build (from package.json next to dist/). */
export const SIDECAR_PACKAGE_VERSION = pkg.version;

/** Sent on every request to `forst dev` so server logs can attribute traffic to a sidecar release. */
export const SIDECAR_VERSION_HTTP_HEADER = "X-Forst-Sidecar-Version";

/**
 * Expected `contractVersion` from `GET /version` for this `@forst/sidecar` release.
 * Must stay in sync with `devHTTPContractVersion` in `forst/cmd/forst/dev_server.go`.
 */
export const FORST_DEV_HTTP_CONTRACT_VERSION = "1";
