import semver from "semver";
import { CompilerBinaryDownloadFailed } from "./errors.js";
import { fetchWithRetry, type FetchImpl } from "./http.js";

const RELEASES_LIST_API =
  "https://api.github.com/repos/forst-lang/forst/releases";

const RELEASE_TAG_API =
  "https://api.github.com/repos/forst-lang/forst/releases/tags";

/** https://docs.github.com/en/rest/about-the-rest-api/api-versions */
const GITHUB_REST_API_VERSION = "2026-03-10";

const GITHUB_JSON_HEADERS = {
  Accept: "application/vnd.github+json",
  "X-GitHub-Api-Version": GITHUB_REST_API_VERSION,
} as const;

/** Compiler release tags only: `v0.4.0`, not `vscode-forst-v…` or `cli-v…`. */
export const COMPILER_RELEASE_TAG_PATTERN =
  /^v(\d+\.\d+\.\d+(?:-[A-Za-z0-9.-]+)?)$/;

/** Why fetchReleaseAssetSha256Hex could not return a digest, surfaced to callers deciding whether to fail strictly. */
export type DigestLookupReason =
  | "release_not_found"
  | "asset_not_found"
  | "digest_missing";

/** Result of a checksum lookup against GitHub release assets: the digest if found, or a reason it wasn't. */
export interface DigestLookupResult {
  hex?: string;
  reason?: DigestLookupReason;
}

/**
 * Parses a GitHub tag into a compiler semver if it is a root compiler release.
 */
export function parseCompilerReleaseVersion(tagName: string): string | undefined {
  const m = tagName.match(COMPILER_RELEASE_TAG_PATTERN);
  return m?.[1];
}

async function fetchCompilerReleaseSemvers(
  fetchImpl: FetchImpl
): Promise<semver.SemVer[]> {
  let page = 1;
  const collected: semver.SemVer[] = [];

  while (page <= 10) {
    const apiUrl = `${RELEASES_LIST_API}?per_page=100&page=${page}`;
    const res = await fetchWithRetry(fetchImpl, apiUrl, {
      headers: GITHUB_JSON_HEADERS,
    });
    if (!res.ok) {
      throw new CompilerBinaryDownloadFailed(
        `GitHub releases list returned HTTP ${res.status} ${res.statusText}`
      );
    }
    const data = (await res.json()) as Array<{ tag_name?: string }>;
    if (!Array.isArray(data) || data.length === 0) {
      break;
    }
    for (const release of data) {
      const version = release.tag_name
        ? parseCompilerReleaseVersion(release.tag_name)
        : undefined;
      if (!version) {
        continue;
      }
      const coerced = semver.coerce(version);
      if (!coerced) {
        continue;
      }
      if (!collected.some((existing) => semver.eq(existing, coerced))) {
        collected.push(coerced);
      }
    }
    if (data.length < 100) {
      break;
    }
    page++;
  }

  collected.sort((a, b) => semver.rcompare(a, b));
  return collected;
}

/**
 * Compiler release semvers (root `vX.Y.Z` tags only), highest first.
 */
export async function fetchCompilerReleaseVersions(
  fetchImpl: FetchImpl = fetch,
  limit = 10
): Promise<string[]> {
  try {
    const collected = await fetchCompilerReleaseSemvers(fetchImpl);
    if (collected.length === 0) {
      throw new CompilerBinaryDownloadFailed(
        "No compiler release tags (vX.Y.Z) found on GitHub"
      );
    }
    return collected.slice(0, limit).map((version) => version.version);
  } catch (e) {
    if (e instanceof CompilerBinaryDownloadFailed) {
      throw e;
    }
    throw new CompilerBinaryDownloadFailed(
      "Failed to fetch Forst compiler release versions",
      { cause: e }
    );
  }
}

/**
 * Semver of the latest published **compiler** release (root `vX.Y.Z` tags only).
 */
export async function fetchLatestCompilerReleaseVersion(
  fetchImpl: FetchImpl = fetch
): Promise<string> {
  try {
    const versions = await fetchCompilerReleaseVersions(fetchImpl, 1);
    return versions[0]!;
  } catch (e) {
    if (e instanceof CompilerBinaryDownloadFailed) {
      throw e;
    }
    throw new CompilerBinaryDownloadFailed(
      "Failed to fetch latest Forst compiler release version",
      { cause: e }
    );
  }
}

function digestLookupReasonMessage(
  reason: DigestLookupReason,
  version: string,
  artifactName: string
): string {
  switch (reason) {
    case "release_not_found":
      return `GitHub release v${version} not found when looking up digest for ${artifactName}`;
    case "asset_not_found":
      return `GitHub release v${version} has no asset named ${artifactName}`;
    case "digest_missing":
      return `GitHub release v${version} asset ${artifactName} has no sha256 digest in API metadata`;
  }
}

/**
 * Returns the sha256 hex digest (no prefix) for a release asset from GitHub
 * release API metadata, plus a reason when lookup fails.
 */
export async function fetchReleaseAssetSha256Hex(
  version: string,
  artifactName: string,
  fetchImpl: FetchImpl
): Promise<DigestLookupResult> {
  const tag = version.startsWith("v") ? version : `v${version}`;
  const apiUrl = `${RELEASE_TAG_API}/${tag}`;
  const res = await fetchWithRetry(fetchImpl, apiUrl, {
    headers: GITHUB_JSON_HEADERS,
  });
  if (!res.ok) {
    return { reason: "release_not_found" };
  }

  const data = (await res.json()) as {
    assets?: Array<{ name: string; digest?: string | null }>;
  };
  const asset = data.assets?.find((a) => a.name === artifactName);
  if (!asset) {
    return { reason: "asset_not_found" };
  }

  const digest = asset.digest;
  if (!digest?.startsWith("sha256:")) {
    return { reason: "digest_missing" };
  }

  return { hex: digest.slice("sha256:".length) };
}

export { digestLookupReasonMessage };
