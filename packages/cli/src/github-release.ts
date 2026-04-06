import { CompilerBinaryDownloadFailed } from "./errors.js";
import { fetchWithRetry } from "./http.js";

const LATEST_RELEASE_API =
  "https://api.github.com/repos/forst-lang/forst/releases/latest";

/**
 * Semver of the latest published GitHub release (e.g. from `releases/latest`).
 */
export async function fetchLatestCompilerReleaseVersion(
  fetchImpl: typeof fetch = fetch
): Promise<string> {
  try {
    const res = await fetchWithRetry(fetchImpl, LATEST_RELEASE_API, {
      headers: {
        Accept: "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
      },
    });
    if (!res.ok) {
      throw new CompilerBinaryDownloadFailed(
        `GitHub releases/latest returned HTTP ${res.status} ${res.statusText}`
      );
    }
    const data = (await res.json()) as { tag_name?: string };
    const tag = data.tag_name;
    if (!tag) {
      throw new CompilerBinaryDownloadFailed(
        "GitHub releases/latest response missing tag_name"
      );
    }
    return tag.replace(/^v/, "");
  } catch (e) {
    if (e instanceof CompilerBinaryDownloadFailed) {
      throw e;
    }
    throw new CompilerBinaryDownloadFailed(
      "Failed to fetch latest Forst release version",
      { cause: e }
    );
  }
}

/**
 * Returns the sha256 hex digest (no prefix) for a release asset, if GitHub
 * exposes it on the API (GitHub attaches `digest` on release assets).
 */
export async function fetchReleaseAssetSha256Hex(
  version: string,
  artifactName: string,
  fetchImpl: typeof fetch
): Promise<string | undefined> {
  const tag = version.startsWith("v") ? version : `v${version}`;
  const apiUrl = `https://api.github.com/repos/forst-lang/forst/releases/tags/${tag}`;
  const res = await fetchWithRetry(fetchImpl, apiUrl, {
    headers: {
      Accept: "application/vnd.github+json",
      "X-GitHub-Api-Version": "2022-11-28",
    },
  });
  if (!res.ok) {
    return undefined;
  }
  const data = (await res.json()) as {
    assets?: Array<{ name: string; digest?: string }>;
  };
  const asset = data.assets?.find((a) => a.name === artifactName);
  const digest = asset?.digest;
  if (!digest?.startsWith("sha256:")) {
    return undefined;
  }
  return digest.slice("sha256:".length);
}
