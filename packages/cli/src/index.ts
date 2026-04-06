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
export { getCliPackageVersion } from "./version.js";
export {
  getCompilerArtifactDownloadUrlForCurrentPlatform,
  getCompilerCacheDirForVersion,
  getExpectedCompilerBinaryPath,
  resolveForstBinary,
  validateCompilerVersionForCachePath,
} from "./resolve.js";
export type {
  ResolveForstBinaryOptions,
  ResolveForstBinaryFs,
} from "./resolve.js";
export {
  CompilerBinaryChecksumMismatch,
  CompilerBinaryDownloadFailed,
  CompilerBinaryDownloadHttpFailure,
  CompilerBinaryNotFound,
  UnsupportedArchitecture,
  UnsupportedOperatingSystem,
} from "./errors.js";
export { fetchLatestCompilerReleaseVersion } from "./github-release.js";
export { fetchWithRetry } from "./http.js";

import { spawn, type SpawnOptions } from "node:child_process";
import { resolveForstBinary, type ResolveForstBinaryOptions } from "./resolve.js";

/**
 * Spawns the resolved native `forst` binary with the given args.
 * Uses `stdio: "inherit"` unless overridden in `spawnOptions`.
 */
export async function spawnForst(
  args: string[],
  spawnOptions: SpawnOptions = {},
  resolveOptions?: ResolveForstBinaryOptions
): Promise<ReturnType<typeof spawn>> {
  const bin = await resolveForstBinary(resolveOptions);
  return spawn(bin, args, { stdio: "inherit", ...spawnOptions });
}
