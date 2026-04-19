import { createHash, randomBytes } from "node:crypto";
import {
  chmodSync,
  existsSync,
  mkdirSync,
  readFileSync,
  renameSync,
  statSync,
  unlinkSync,
  writeFileSync,
} from "node:fs";
import { open } from "node:fs/promises";
import { homedir } from "node:os";
import { dirname, join } from "node:path";
import semver from "semver";
import { getCompilerArtifactName } from "./artifact.js";
import { COMPILER_RELEASES_BASE } from "./constants.js";
import {
  CompilerBinaryChecksumMismatch,
  CompilerBinaryDownloadFailed,
  CompilerBinaryDownloadHttpFailure,
  CompilerBinaryNotFound,
} from "./errors.js";
import { fetchReleaseAssetSha256Hex } from "./github-release.js";
import { fetchWithRetry } from "./http.js";
import { buildCompilerArtifactDownloadUrl } from "./urls.js";
import { getCliPackageVersion } from "./version.js";

/** Download URL for the compiler artifact matching this @forst/cli version and current OS/arch. */
export function getCompilerArtifactDownloadUrlForCurrentPlatform(
  platform: NodeJS.Platform = process.platform,
  archName: string = process.arch
): string {
  const version = getCliPackageVersion();
  const artifact = getCompilerArtifactName(platform, archName);
  return buildCompilerArtifactDownloadUrl(version, artifact);
}

const STALE_LOCK_MS = 10 * 60 * 1000;
const LOCK_WAIT_MS = 120_000;
const POLL_MS = 200;

/** How long to reuse GitHub `releases/latest` when {@link ResolveForstBinaryOptions.preferLatestRelease} is set. */
const LATEST_RELEASE_CACHE_TTL_MS = 60 * 60 * 1000;
const LATEST_RELEASE_CACHE_FILENAME = ".latest-compiler-release.json";

/** Semver-ish release id allowed as a single cache path segment (no separators). */
const COMPILER_VERSION_FOR_CACHE_PATTERN =
  /^[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?$/;

/**
 * Validates `version` before using it in {@link join} under FORST_CACHE_DIR or
 * default cache roots so values cannot escape the intended directory.
 */
export function validateCompilerVersionForCachePath(version: string): string {
  const v = version.trim();
  if (!COMPILER_VERSION_FOR_CACHE_PATTERN.test(v)) {
    throw new CompilerBinaryDownloadFailed(
      `Invalid Forst compiler version "${version}". Expected semver such as 0.0.19 (optional prerelease suffix).`
    );
  }
  return v;
}

export type ResolveForstBinaryFs = Pick<
  typeof import("node:fs"),
  | "existsSync"
  | "mkdirSync"
  | "readFileSync"
  | "writeFileSync"
  | "chmodSync"
  | "renameSync"
  | "unlinkSync"
  | "statSync"
>;

export interface ResolveForstBinaryOptions {
  /** Compiler version to fetch (default: @forst/cli package version). */
  version?: string;
  /**
   * When true and {@link allowDownload} is true, fetches the latest GitHub release tag and uses the
   * higher of that and the bundled @forst/cli semver for the cache path (may download a newer binary).
   * Ignored when {@link version} is set. Default false for API callers; the VS Code extension defaults this on.
   */
  preferLatestRelease?: boolean;
  /**
   * When false, only `FORST_BINARY` or an existing cached binary is used; no network download.
   * Default true (download if missing).
   */
  allowDownload?: boolean;
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
  const v = validateCompilerVersionForCachePath(version);
  const env = options?.env ?? process.env;
  const home = options?.homedirFn ?? homedir;

  if (env.FORST_CACHE_DIR && env.FORST_CACHE_DIR.length > 0) {
    return join(env.FORST_CACHE_DIR, v);
  }

  if (process.platform === "win32") {
    const base =
      env.LOCALAPPDATA ?? join(home(), "AppData", "Local");
    return join(base, "forst-cli", "cache", v);
  }

  const xdg = env.XDG_CACHE_HOME;
  if (xdg && xdg.length > 0) {
    return join(xdg, "forst-cli", v);
  }

  return join(home(), ".cache", "forst-cli", v);
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

/** Root under which per-version compiler directories are stored (e.g. `~/.cache/forst-cli`). */
export function getCompilerCacheBaseDir(
  options?: { env?: NodeJS.ProcessEnv; homedirFn?: () => string }
): string {
  return dirname(getCompilerCacheDirForVersion("0.0.0", options));
}

/**
 * Returns the higher of two semver strings (using `semver.coerce`). Used for bundled vs GitHub latest.
 */
export function maxSemverCompilerVersion(a: string, b: string): string {
  const va = semver.coerce(a);
  const vb = semver.coerce(b);
  if (!va || !vb) {
    throw new CompilerBinaryDownloadFailed(
      `Invalid semver for compiler version comparison: "${a}" vs "${b}"`
    );
  }
  return semver.gt(va, vb) ? a.trim() : b.trim();
}

interface LatestReleaseDiskCache {
  version: string;
  fetchedAt: number;
}

async function fetchLatestReleaseVersionWithDiskCache(
  fetchFn: typeof fetch,
  env: NodeJS.ProcessEnv,
  homedirFn: () => string,
  fs: ResolveForstBinaryFs
): Promise<string> {
  const base = getCompilerCacheBaseDir({ env, homedirFn });
  const cachePath = join(base, LATEST_RELEASE_CACHE_FILENAME);
  try {
    if (fs.existsSync(cachePath)) {
      const raw = fs.readFileSync(cachePath, "utf8");
      const parsed = JSON.parse(raw) as LatestReleaseDiskCache;
      if (
        typeof parsed.version === "string" &&
        typeof parsed.fetchedAt === "number" &&
        Date.now() - parsed.fetchedAt < LATEST_RELEASE_CACHE_TTL_MS
      ) {
        return parsed.version;
      }
    }
  } catch {
    /* ignore */
  }

  const { fetchLatestCompilerReleaseVersion } = await import("./github-release.js");
  const version = await fetchLatestCompilerReleaseVersion(fetchFn);
  try {
    fs.mkdirSync(base, { recursive: true });
    fs.writeFileSync(
      cachePath,
      JSON.stringify({
        version,
        fetchedAt: Date.now(),
      } satisfies LatestReleaseDiskCache),
      "utf8"
    );
  } catch {
    /* ignore */
  }
  return version;
}

async function resolveEffectiveCompilerVersion(
  options: ResolveForstBinaryOptions,
  allowDownload: boolean,
  fs: ResolveForstBinaryFs
): Promise<string> {
  if (options.version !== undefined && options.version.length > 0) {
    return validateCompilerVersionForCachePath(options.version);
  }

  const bundled = getCliPackageVersion();
  const prefer =
    options.preferLatestRelease === true && allowDownload;
  if (!prefer) {
    return validateCompilerVersionForCachePath(bundled);
  }

  const fetchFn = options.fetchImpl ?? fetch;
  const env = options.env ?? process.env;
  const homedirFn = options.homedirFn ?? homedir;

  try {
    const latest = await fetchLatestReleaseVersionWithDiskCache(
      fetchFn,
      env,
      homedirFn,
      fs
    );
    try {
      const picked = maxSemverCompilerVersion(bundled, latest);
      return validateCompilerVersionForCachePath(picked);
    } catch {
      return validateCompilerVersionForCachePath(bundled);
    }
  } catch {
    return validateCompilerVersionForCachePath(bundled);
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function acquireLockOrWaitForBinary(
  dest: string,
  lockPath: string,
  fs: ResolveForstBinaryFs
): Promise<{ release: () => Promise<void>; skippedDownload: boolean }> {
  fs.mkdirSync(dirname(dest), { recursive: true });

  const deadline = Date.now() + LOCK_WAIT_MS;

  while (Date.now() < deadline) {
    if (fs.existsSync(dest)) {
      return { release: async () => { }, skippedDownload: true };
    }

    if (fs.existsSync(lockPath)) {
      try {
        const age = Date.now() - fs.statSync(lockPath).mtimeMs;
        if (age > STALE_LOCK_MS) {
          fs.unlinkSync(lockPath);
        }
      } catch {
        /* ignore */
      }
      await sleep(POLL_MS);
      continue;
    }

    try {
      const fh = await open(lockPath, "wx");
      return {
        skippedDownload: false,
        release: async () => {
          await fh.close();
          try {
            const { unlink } = await import("node:fs/promises");
            await unlink(lockPath);
          } catch {
            /* ignore */
          }
        },
      };
    } catch (e: unknown) {
      const err = e as NodeJS.ErrnoException;
      if (err.code !== "EEXIST") {
        throw e;
      }
      await sleep(POLL_MS);
    }
  }

  if (fs.existsSync(dest)) {
    return { release: async () => { }, skippedDownload: true };
  }

  throw new CompilerBinaryDownloadFailed(
    "Timed out waiting for concurrent Forst compiler download"
  );
}

function verifySha256(buf: Buffer, expectedHex: string): void {
  const actual = createHash("sha256").update(buf).digest("hex");
  if (actual !== expectedHex) {
    throw new CompilerBinaryChecksumMismatch(expectedHex, actual);
  }
}

function writeBinaryAtomically(
  dest: string,
  buf: Buffer,
  fs: ResolveForstBinaryFs
): void {
  const part = `${dest}.part.${randomBytes(8).toString("hex")}`;
  try {
    fs.mkdirSync(dirname(dest), { recursive: true });
    fs.writeFileSync(part, buf);
    fs.renameSync(part, dest);
    if (process.platform !== "win32") {
      fs.chmodSync(dest, 0o755);
    }
  } catch (e) {
    try {
      fs.unlinkSync(part);
    } catch {
      /* ignore */
    }
    throw new CompilerBinaryDownloadFailed(
      `Failed to install Forst compiler to ${dest}`,
      { cause: e }
    );
  }
}

/**
 * Resolves the native `forst` executable: `FORST_BINARY` if set, otherwise a
 * cached binary downloaded from GitHub Releases for this package version.
 */
export async function resolveForstBinary(
  options: ResolveForstBinaryOptions = {}
): Promise<string> {
  const allowDownload = options.allowDownload !== false;
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
    readFileSync,
    writeFileSync,
    chmodSync,
    renameSync,
    unlinkSync,
    statSync,
  };
  const version = await resolveEffectiveCompilerVersion(options, allowDownload, fs);
  const artifact = getCompilerArtifactName(process.platform, process.arch);
  const dest = getExpectedCompilerBinaryPath(version, process.platform, process.arch, {
    env,
    homedirFn: options.homedirFn,
  });

  if (fs.existsSync(dest)) {
    return dest;
  }

  if (!allowDownload) {
    throw new CompilerBinaryNotFound(
      `Forst compiler is not installed in the local cache (${dest}). ` +
        `Set FORST_BINARY to the executable, enable downloads, or install the @forst/cli cache.`
    );
  }

  const url = buildCompilerArtifactDownloadUrl(version, artifact);
  const fetchFn = options.fetchImpl ?? fetch;
  const lockPath = `${dest}.lock`;

  const { release, skippedDownload } = await acquireLockOrWaitForBinary(
    dest,
    lockPath,
    fs
  );

  try {
    if (skippedDownload || fs.existsSync(dest)) {
      return dest;
    }

    const verificationDisabled =
      env.FORST_CLI_VERIFY === "0" || env.FORST_CLI_VERIFY === "false";

    let expectedDigest: string | undefined;
    if (!verificationDisabled) {
      const hex = await fetchReleaseAssetSha256Hex(
        version,
        artifact,
        fetchFn
      );
      if (!hex) {
        throw new CompilerBinaryDownloadFailed(
          `Cannot verify the Forst compiler download: GitHub did not provide a sha256 digest for ${artifact} (release v${version}). ` +
            `Refusing to install an unverified binary. Set FORST_CLI_VERIFY=0 to skip verification.`
        );
      }
      expectedDigest = hex;
    }

    let response: Response;
    try {
      response = await fetchWithRetry(fetchFn, url);
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

    if (!verificationDisabled) {
      verifySha256(buf, expectedDigest!);
    }

    writeBinaryAtomically(dest, buf, fs);
    return dest;
  } finally {
    await release();
  }
}
