export { ForstSidecarClient } from "./client";
export {
  ForstServer,
  buildForstDevSpawnArgs,
  buildForstWatchRoots,
  effectiveProjectRootDir,
  effectiveWatchDirForConfig,
} from "./server";
export { ForstUtils } from "./utils";
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
  autoStart,
} from "./sidecar";
export { default } from "./sidecar";
