export { COMPILER_RELEASES_BASE } from "./constants.js";
export {
  getCompilerArtifactName,
} from "./artifact.js";
export {
  buildCompilerArtifactDownloadUrl,
} from "./urls.js";
export { getCliPackageVersion } from "./version.js";
export {
  getCompilerCacheDirForVersion,
  getExpectedCompilerBinaryPath,
  resolveForstBinary,
} from "./resolve.js";
export type {
  ResolveForstBinaryOptions,
  ResolveForstBinaryFs,
} from "./resolve.js";
export {
  CompilerBinaryDownloadFailed,
  CompilerBinaryDownloadHttpFailure,
  UnsupportedArchitecture,
  UnsupportedOperatingSystem,
} from "./errors.js";

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
