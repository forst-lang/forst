export {
  COMPILER_RELEASES_BASE,
  FORST_CLI_INFO_FLAG,
  FORST_CLI_GO_BUILDINFO_FLAG,
  FORST_CLI_VERSION_FLAGS,
} from "./constants.js";
export {
  printForstCliInfo,
  printForstCliVersion,
  printForstGoBuildInfo,
} from "./cli-info.js";
export {
  getCompilerArtifactName,
} from "./artifact.js";
export {
  buildCompilerArtifactDownloadUrl,
} from "./urls.js";
export { getCliPackageVersion, getBundledCompilerReleaseVersion } from "./version.js";
export {
  getCompilerArtifactDownloadUrlForCurrentPlatform,
  getCompilerCacheBaseDir,
  getCompilerCacheDirForVersion,
  getExpectedCompilerBinaryPath,
  maxSemverCompilerVersion,
  resolveForstBinary,
  resolveForstBinaryDetailed,
  validateCompilerVersionForCachePath,
} from "./resolve.js";
export type {
  ResolveForstBinaryOptions,
  ResolveForstBinaryFs,
  ResolvedForstBinary,
} from "./resolve.js";
export { buildForstSpawnEnv } from "./spawn-env.js";
export {
  getCompilerModuleDirForVersion,
  ensureCompilerModuleForVersion,
} from "./compiler-module.js";
export {
  CompilerBinaryChecksumMismatch,
  CompilerBinaryDigestUnavailable,
  CompilerBinaryDownloadFailed,
  CompilerBinaryDownloadHttpFailure,
  CompilerBinaryNotFound,
  UnsupportedArchitecture,
  UnsupportedOperatingSystem,
} from "./errors.js";
export { fetchLatestCompilerReleaseVersion } from "./github-release.js";
export { fetchWithRetry } from "./http.js";

import { spawn, type SpawnOptions } from "node:child_process";
import { buildForstSpawnEnv } from "./spawn-env.js";
import { type ResolveForstBinaryOptions } from "./resolve.js";

/**
 * Spawns the resolved native `forst` binary with the given args.
 * Uses `stdio: "inherit"` unless overridden in `spawnOptions`.
 */
export async function spawnForst(
  args: string[],
  spawnOptions: SpawnOptions = {},
  resolveOptions?: ResolveForstBinaryOptions
): Promise<ReturnType<typeof spawn>> {
  const { bin, env } = await buildForstSpawnEnv(resolveOptions);
  return spawn(bin, args, {
    stdio: "inherit",
    ...spawnOptions,
    env: { ...env, ...spawnOptions.env },
  });
}
