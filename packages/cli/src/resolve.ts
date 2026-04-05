import { chmodSync, existsSync, mkdirSync, writeFileSync } from "node:fs";
import { homedir } from "node:os";
import { dirname, join } from "node:path";
import { getCompilerArtifactName } from "./artifact.js";
import { COMPILER_RELEASES_BASE } from "./constants.js";
import {
  CompilerBinaryDownloadFailed,
  CompilerBinaryDownloadHttpFailure,
} from "./errors.js";
import { buildCompilerArtifactDownloadUrl } from "./urls.js";
import { getCliPackageVersion } from "./version.js";

export type ResolveForstBinaryFs = Pick<
  typeof import("node:fs"),
  "existsSync" | "mkdirSync" | "writeFileSync" | "chmodSync"
>;

export interface ResolveForstBinaryOptions {
  /** Compiler version to fetch (default: @forst/cli package version). */
  version?: string;
  /** Override path to the native binary (skips download). */
  env?: NodeJS.ProcessEnv;
  fetchImpl?: typeof fetch;
  fs?: ResolveForstBinaryFs;
  homedirFn?: () => string;
}

/** Root cache directory for one compiler version (no trailing artifact name). */
export function getCompilerCacheDirForVersion(
  version: string,
  options?: { env?: NodeJS.ProcessEnv; homedirFn?: () => string }
): string {
  const env = options?.env ?? process.env;
  const home = options?.homedirFn ?? homedir;

  if (env.FORST_CACHE_DIR && env.FORST_CACHE_DIR.length > 0) {
    return join(env.FORST_CACHE_DIR, version);
  }

  if (process.platform === "win32") {
    const base =
      env.LOCALAPPDATA ?? join(home(), "AppData", "Local");
    return join(base, "forst-cli", "cache", version);
  }

  const xdg = env.XDG_CACHE_HOME;
  if (xdg && xdg.length > 0) {
    return join(xdg, "forst-cli", version);
  }

  return join(home(), ".cache", "forst-cli", version);
}

/** Path where the binary for this version/platform would be stored (may not exist yet). */
export function getExpectedCompilerBinaryPath(
  version: string,
  platform: NodeJS.Platform = process.platform,
  arch: string = process.arch,
  options?: { env?: NodeJS.ProcessEnv; homedirFn?: () => string }
): string {
  const artifact = getCompilerArtifactName(platform, arch);
  return join(getCompilerCacheDirForVersion(version, options), artifact);
}

/**
 * Resolves the native `forst` executable: `FORST_BINARY` if set, otherwise a
 * cached binary downloaded from GitHub Releases for this package version.
 */
export async function resolveForstBinary(
  options: ResolveForstBinaryOptions = {}
): Promise<string> {
  const env = options.env ?? process.env;
  const override = env.FORST_BINARY?.trim();
  if (override) {
    if (!existsSync(override)) {
      throw new CompilerBinaryDownloadFailed(
        `FORST_BINARY is set to "${override}" but that file does not exist`
      );
    }
    return override;
  }

  const fs = options.fs ?? {
    existsSync,
    mkdirSync,
    writeFileSync,
    chmodSync,
  };
  const version = options.version ?? getCliPackageVersion();
  const artifact = getCompilerArtifactName(process.platform, process.arch);
  const dest = getExpectedCompilerBinaryPath(version, process.platform, process.arch, {
    env,
    homedirFn: options.homedirFn,
  });

  if (fs.existsSync(dest)) {
    return dest;
  }

  const url = buildCompilerArtifactDownloadUrl(version, artifact);
  const fetchFn = options.fetchImpl ?? fetch;

  let response: Awaited<ReturnType<typeof fetch>>;
  try {
    response = await fetchFn(url);
  } catch (e) {
    throw new CompilerBinaryDownloadFailed(
      `Failed to download Forst compiler from ${url} (${COMPILER_RELEASES_BASE})`,
      { cause: e }
    );
  }

  if (!response.ok) {
    throw new CompilerBinaryDownloadHttpFailure(
      response.status,
      response.statusText
    );
  }

  const buf = Buffer.from(await response.arrayBuffer());
  fs.mkdirSync(dirname(dest), { recursive: true });
  try {
    fs.writeFileSync(dest, buf);
    if (process.platform !== "win32") {
      fs.chmodSync(dest, 0o755);
    }
  } catch (e) {
    throw new CompilerBinaryDownloadFailed(
      `Failed to write Forst compiler to ${dest}`,
      { cause: e }
    );
  }

  return dest;
}
