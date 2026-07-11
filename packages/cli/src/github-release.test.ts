import { describe, expect, test } from "bun:test";
import {
  COMPILER_RELEASE_TAG_PATTERN,
  fetchCompilerReleaseVersions,
  fetchLatestCompilerReleaseVersion,
  fetchReleaseAssetSha256Hex,
  parseCompilerReleaseVersion,
} from "./github-release.js";
import type { FetchImpl } from "./http.js";

test("parseCompilerReleaseVersion accepts root compiler tags only", () => {
  expect(parseCompilerReleaseVersion("v0.4.0")).toBe("0.4.0");
  expect(parseCompilerReleaseVersion("v1.2.3-rc.1")).toBe("1.2.3-rc.1");
  expect(parseCompilerReleaseVersion("vscode-forst-v0.3.0")).toBeUndefined();
  expect(parseCompilerReleaseVersion("cli-v0.0.35")).toBeUndefined();
});

test("COMPILER_RELEASE_TAG_PATTERN matches v-prefixed semver tags", () => {
  expect(COMPILER_RELEASE_TAG_PATTERN.test("v0.4.0")).toBe(true);
  expect(COMPILER_RELEASE_TAG_PATTERN.test("vscode-forst-v0.3.0")).toBe(false);
});

describe("fetchCompilerReleaseVersions", () => {
  test("returns compiler tags sorted highest first", async () => {
    const fetchImpl: FetchImpl = async (url) => {
      const s = String(url);
      if (s.includes("/repos/forst-lang/forst/releases?")) {
        return new Response(
          JSON.stringify([
            { tag_name: "vscode-forst-v0.3.0" },
            { tag_name: "v0.4.0" },
            { tag_name: "v0.4.1" },
            { tag_name: "v0.0.35" },
            { tag_name: "v0.10.0" },
          ]),
          { status: 200, headers: { "Content-Type": "application/json" } }
        );
      }
      throw new Error(`unexpected fetch: ${s}`);
    };

    const versions = await fetchCompilerReleaseVersions(fetchImpl);
    expect(versions).toEqual([
      "0.10.0",
      "0.4.1",
      "0.4.0",
      "0.0.35",
    ]);
  });
});

describe("fetchLatestCompilerReleaseVersion", () => {
  test("picks highest vX.Y.Z tag and ignores vscode/cli tags", async () => {
    const fetchImpl: FetchImpl = async (url) => {
      const s = String(url);
      if (s.includes("/repos/forst-lang/forst/releases?")) {
        return new Response(
          JSON.stringify([
            { tag_name: "vscode-forst-v0.3.0" },
            { tag_name: "cli-v0.0.35" },
            { tag_name: "v0.4.0" },
            { tag_name: "v0.0.35" },
          ]),
          { status: 200, headers: { "Content-Type": "application/json" } }
        );
      }
      throw new Error(`unexpected fetch: ${s}`);
    };

    const version = await fetchLatestCompilerReleaseVersion(fetchImpl);
    expect(version).toBe("0.4.0");
  });
});

describe("fetchReleaseAssetSha256Hex", () => {
  test("returns hex when GitHub API provides digest", async () => {
    const fetchImpl: FetchImpl = async () =>
      new Response(
        JSON.stringify({
          assets: [{ name: "forst-darwin-arm64", digest: "sha256:" + "ab".repeat(32) }],
        }),
        { status: 200, headers: { "Content-Type": "application/json" } }
      );

    const result = await fetchReleaseAssetSha256Hex(
      "0.4.0",
      "forst-darwin-arm64",
      fetchImpl
    );
    expect(result.hex).toBe("ab".repeat(32));
    expect(result.reason).toBeUndefined();
  });

  test("returns asset_not_found when release exists but asset is missing", async () => {
    const fetchImpl: FetchImpl = async () =>
      new Response(
        JSON.stringify({ assets: [{ name: "other-asset" }] }),
        { status: 200, headers: { "Content-Type": "application/json" } }
      );

    const result = await fetchReleaseAssetSha256Hex(
      "0.4.0",
      "forst-darwin-arm64",
      fetchImpl
    );
    expect(result.hex).toBeUndefined();
    expect(result.reason).toBe("asset_not_found");
  });

  test("returns release_not_found when tag is missing", async () => {
    const fetchImpl: FetchImpl = async () =>
      new Response(null, { status: 404, statusText: "Not Found" });

    const result = await fetchReleaseAssetSha256Hex(
      "9.9.9",
      "forst-darwin-arm64",
      fetchImpl
    );
    expect(result.hex).toBeUndefined();
    expect(result.reason).toBe("release_not_found");
  });
});
