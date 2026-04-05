import { fetchWithRetry } from "./http.js";

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
