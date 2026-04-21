export { ForstSidecarClient } from "./client";
export {
  ForstServer,
  buildForstDevSpawnArgs,
  buildForstGenerateArgs,
  buildForstWatchRoots,
  effectiveProjectRootDir,
  effectiveWatchDirForConfig,
} from "./server";
export { ForstUtils } from "./utils";
export {
  FORST_DEV_HTTP_CONTRACT_VERSION,
  SIDECAR_PACKAGE_VERSION,
  SIDECAR_VERSION_HTTP_HEADER,
} from "./constants";
export {
  contractVersionCompatible,
  versionsEquivalentForSidecar,
} from "./version-compare";
export {
  mergeForstSidecarEnv,
  normalizeDevServerBaseUrl,
  parseDevServerUrlParts,
} from "./config-merge";
export * from "./types";
export * from "./errors";

export {
  ForstSidecar,
  createExpressMiddleware,
  createRouteToForstMiddleware,
  autoStart,
} from "./sidecar";
export { default } from "./sidecar";
