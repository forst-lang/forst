import { createHash } from "node:crypto";
import { describe, expect, test } from "bun:test";
import {
  chmodSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { getCompilerArtifactName } from "./artifact.js";
import { buildCompilerArtifactDownloadUrl } from "./urls.js";
import type { FetchImpl } from "./http.js";
import {
  maxSemverCompilerVersion,
  resolveFallbackCompilerVersion,
  resolveForstBinary,
} from "./resolve.js";
import { getBundledCompilerReleaseVersion } from "./version.js";
import {
  CompilerBinaryChecksumMismatch,
  CompilerBinaryDigestUnavailable,
  CompilerBinaryDownloadFailed,
  CompilerBinaryDownloadHttpFailure,
  CompilerBinaryNotFound,
} from "./errors.js";

const verifyOff = { FORST_CLI_VERIFY: "0" } as const;

function seedCompilerModuleCache(cacheRoot: string, version: string): void {
  mkdirSync(join(cacheRoot, version, "module", "cmd", "forst"), {
    recursive: true,
  });
}

test("getCompilerArtifactName matches release task naming", () => {
  expect(getCompilerArtifactName("darwin", "arm64")).toBe("forst-darwin-arm64");
  expect(getCompilerArtifactName("linux", "x64")).toBe("forst-linux-amd64");
  expect(getCompilerArtifactName("win32", "x64")).toBe("forst-windows-amd64.exe");
});

test("buildCompilerArtifactDownloadUrl", () => {
  expect(
    buildCompilerArtifactDownloadUrl("0.6.0", "forst-darwin-arm64")
  ).toBe(
    "https://github.com/forst-lang/forst/releases/download/v0.6.0/forst-darwin-arm64"
  );
});

test("maxSemverCompilerVersion picks the higher semver", () => {
  expect(maxSemverCompilerVersion("0.0.24", "0.0.34")).toBe("0.0.34");
  expect(maxSemverCompilerVersion("1.0.0", "0.9.9")).toBe("1.0.0");
  expect(maxSemverCompilerVersion("0.0.10", "0.0.9")).toBe("0.0.10");
});

describe("resolveForstBinary", () => {
  test("uses FORST_BINARY when set and file exists", async () => {
    const dir = mkdtempSync(join(tmpdir(), "forst-cli-test-"));
    try {
      const fake = join(dir, "forst-fake");
      writeFileSync(fake, "#!/bin/sh\necho ok\n");
      if (process.platform !== "win32") {
        chmodSync(fake, 0o755);
      }
      const p = await resolveForstBinary({
        env: { ...process.env, FORST_BINARY: fake },
      });
      expect(p).toBe(fake);
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  });

  test("throws when FORST_BINARY points at missing file", async () => {
    const err = await resolveForstBinary({
      env: { ...process.env, FORST_BINARY: "/nonexistent/forst-binary-xyz" },
    }).catch((e) => e);

    expect(String(err)).toMatch(/FORST_BINARY/);
  });

  test("allowDownload false throws when cache miss and no FORST_BINARY", async () => {
    const cacheRoot = mkdtempSync(join(tmpdir(), "forst-cli-cache-"));
    try {
      const err = await resolveForstBinary({
        version: "0.6.0",
        allowDownload: false,
        env: {
          ...process.env,
          FORST_CACHE_DIR: cacheRoot
        },
        homedirFn: () => "/unused",
      }).catch((e) => e);

      expect(err).toBeInstanceOf(CompilerBinaryNotFound);
    } finally {
      rmSync(cacheRoot, { recursive: true, force: true });
    }
  });

  test("downloads into cache when missing", async () => {
    const cacheRoot = mkdtempSync(join(tmpdir(), "forst-cli-cache-"));
    try {
      const destName =
        process.platform === "win32"
          ? "forst-windows-amd64.exe"
          : getCompilerArtifactName(process.platform, process.arch);
      const dest = join(cacheRoot, "0.6.0", destName);

      const fakeBinary = Buffer.from("fake-forst-binary");
      const fetchImpl: FetchImpl = async (url) => {
        expect(String(url)).toContain("/download/v0.6.0/" + destName);
        return new Response(fakeBinary, { status: 200 });
      };
      seedCompilerModuleCache(cacheRoot, "0.6.0");

      const p = await resolveForstBinary({
        version: "0.6.0",
        env: { ...process.env, ...verifyOff, FORST_CACHE_DIR: cacheRoot },
        fetchImpl,
        homedirFn: () => "/unused",
      });

      expect(p).toBe(dest);
      expect(readFileSync(dest).equals(fakeBinary)).toBe(true);
    } finally {
      rmSync(cacheRoot, { recursive: true, force: true });
    }
  });

  test("cache hit skips download and does not call fetchImpl", async () => {
    const cacheRoot = mkdtempSync(join(tmpdir(), "forst-cli-cache-"));
    try {
      const destName =
        process.platform === "win32"
          ? "forst-windows-amd64.exe"
          : getCompilerArtifactName(process.platform, process.arch);
      const dest = join(cacheRoot, "0.6.0", destName);
      mkdirSync(dirname(dest), { recursive: true });
      const marker = Buffer.from("cached-binary-contents");
      writeFileSync(dest, marker);
      if (process.platform !== "win32") {
        chmodSync(dest, 0o755);
      }
      seedCompilerModuleCache(cacheRoot, "0.6.0");

      let fetchCalled = false;
      const fetchImpl: FetchImpl = async () => {
        fetchCalled = true;
        throw new Error("fetchImpl must not be called on cache hit");
      };

      const p = await resolveForstBinary({
        version: "0.6.0",
        env: {
          ...process.env,
          ...verifyOff,
          FORST_CACHE_DIR: cacheRoot
        },
        fetchImpl,
        homedirFn: () => "/unused",
      });

      expect(p).toBe(dest);
      expect(readFileSync(dest).equals(marker)).toBe(true);
      expect(fetchCalled).toBe(false);
    } finally {
      rmSync(cacheRoot, { recursive: true, force: true });
    }
  });

  test("rejects invalid version strings used for cache paths", async () => {
    const err = await resolveForstBinary({
      version: "../../../evil",
      env: { ...process.env, FORST_CACHE_DIR: "/tmp" },
      homedirFn: () => "/unused",
    }).catch((e) => e);

    expect(String(err)).toMatch(/Invalid Forst compiler version/);
  });

  test("falls back to previous release when latest binary returns 404", async () => {
    const cacheRoot = mkdtempSync(join(tmpdir(), "forst-cli-cache-"));
    try {
      const destName =
        process.platform === "win32"
          ? "forst-windows-amd64.exe"
          : getCompilerArtifactName(process.platform, process.arch);
      const latest = getBundledCompilerReleaseVersion();
      const fallback = "0.0.18";
      const dest = join(cacheRoot, fallback, destName);
      const fakeBinary = Buffer.from("fallback-forst-binary");

      const fetchImpl: FetchImpl = async (url) => {
        const s = String(url);
        if (s.includes("/repos/forst-lang/forst/releases?")) {
          return new Response(
            JSON.stringify([
              { tag_name: `v${latest}` },
              { tag_name: `v${fallback}` },
            ]),
            { status: 200, headers: { "Content-Type": "application/json" } }
          );
        }
        if (s.includes(`/download/v${latest}/`)) {
          return new Response(null, { status: 404, statusText: "Not Found" });
        }
        if (s.includes(`/download/v${fallback}/`)) {
          return new Response(fakeBinary, { status: 200 });
        }
        if (s.includes("api.github.com")) {
          return new Response(
            JSON.stringify({ assets: [{ name: destName }] }),
            { status: 200, headers: { "Content-Type": "application/json" } }
          );
        }
        throw new Error(`unexpected fetch: ${s}`);
      };
      seedCompilerModuleCache(cacheRoot, fallback);

      const p = await resolveForstBinary({
        env: { ...process.env, ...verifyOff, FORST_CACHE_DIR: cacheRoot },
        fetchImpl,
        homedirFn: () => "/unused",
      });

      expect(p).toBe(dest);
      expect(readFileSync(dest).equals(fakeBinary)).toBe(true);
    } finally {
      rmSync(cacheRoot, { recursive: true, force: true });
    }
  });

  test("falls back to latest published release when bundled version is ahead of GitHub", async () => {
    const cacheRoot = mkdtempSync(join(tmpdir(), "forst-cli-cache-"));
    try {
      const destName =
        process.platform === "win32"
          ? "forst-windows-amd64.exe"
          : getCompilerArtifactName(process.platform, process.arch);
      const bundled = getBundledCompilerReleaseVersion();
      const published = "0.0.18";
      const dest = join(cacheRoot, published, destName);
      const fakeBinary = Buffer.from("published-forst-binary");

      const fetchImpl: FetchImpl = async (url) => {
        const s = String(url);
        if (s.includes("/repos/forst-lang/forst/releases?")) {
          return new Response(
            JSON.stringify([{ tag_name: `v${published}` }]),
            { status: 200, headers: { "Content-Type": "application/json" } }
          );
        }
        if (s.includes(`/download/v${bundled}/`)) {
          return new Response(null, { status: 404, statusText: "Not Found" });
        }
        if (s.includes(`/download/v${published}/`)) {
          return new Response(fakeBinary, { status: 200 });
        }
        if (s.includes("api.github.com")) {
          return new Response(
            JSON.stringify({ assets: [{ name: destName }] }),
            { status: 200, headers: { "Content-Type": "application/json" } }
          );
        }
        throw new Error(`unexpected fetch: ${s}`);
      };
      seedCompilerModuleCache(cacheRoot, published);

      const p = await resolveForstBinary({
        env: { ...process.env, ...verifyOff, FORST_CACHE_DIR: cacheRoot },
        fetchImpl,
        homedirFn: () => "/unused",
      });

      expect(p).toBe(dest);
      expect(readFileSync(dest).equals(fakeBinary)).toBe(true);
    } finally {
      rmSync(cacheRoot, { recursive: true, force: true });
    }
  });

  test("resolveFallbackCompilerVersion returns second latest when failed version is latest", async () => {
    const fetchImpl: FetchImpl = async (url) => {
      if (String(url).includes("/repos/forst-lang/forst/releases?")) {
        return new Response(
          JSON.stringify([
            { tag_name: "v0.5.3" },
            { tag_name: "v0.5.2" },
          ]),
          { status: 200, headers: { "Content-Type": "application/json" } }
        );
      }
      throw new Error(`unexpected fetch: ${String(url)}`);
    };

    const version = await resolveFallbackCompilerVersion("0.5.3", fetchImpl);
    expect(version).toBe("0.5.2");
  });

  test("maps HTTP failure to CompilerBinaryDownloadHttpFailure", async () => {
    const cacheRoot = mkdtempSync(join(tmpdir(), "forst-cli-cache-"));
    try {
      const fetchImpl: FetchImpl = async () =>
        new Response(null, { status: 404, statusText: "Not Found" });

      const err = await resolveForstBinary({
        version: "0.6.0",
        env: {
          ...process.env,
          ...verifyOff,
          FORST_CACHE_DIR: cacheRoot
        },
        fetchImpl,
        homedirFn: () => "/unused",
      }).catch((e) => e);

      expect(err).toBeInstanceOf(CompilerBinaryDownloadHttpFailure);
    } finally {
      rmSync(cacheRoot, { recursive: true, force: true });
    }
  });

  test("verifies sha256 when GitHub API returns digest", async () => {
    const cacheRoot = mkdtempSync(join(tmpdir(), "forst-cli-cache-"));
    try {
      const destName =
        process.platform === "win32"
          ? "forst-windows-amd64.exe"
          : getCompilerArtifactName(process.platform, process.arch);
      const dest = join(cacheRoot, "0.6.0", destName);
      const payload = Buffer.from("fake-forst-binary");
      const hex = createHash("sha256").update(payload).digest("hex");

      const fetchImpl: FetchImpl = async (url) => {
        const s = String(url);
        if (s.includes("api.github.com")) {
          return new Response(
            JSON.stringify({
              assets: [{
                name: destName,
                digest: `sha256:${hex}`
              }],
            }),
            { status: 200, headers: { "Content-Type": "application/json" } }
          );
        }
        expect(s).toContain("/download/v0.6.0/" + destName);
        return new Response(payload, { status: 200 });
      };
      seedCompilerModuleCache(cacheRoot, "0.6.0");

      const p = await resolveForstBinary({
        version: "0.6.0",
        env: {
          ...process.env,
          FORST_CACHE_DIR: cacheRoot
        },
        fetchImpl,
        homedirFn: () => "/unused",
      });

      expect(p).toBe(dest);
    } finally {
      rmSync(cacheRoot, { recursive: true, force: true });
    }
  });

  test("downloads without verification when GitHub API returns no digest", async () => {
    const cacheRoot = mkdtempSync(join(tmpdir(), "forst-cli-cache-"));
    try {
      const destName =
        process.platform === "win32"
          ? "forst-windows-amd64.exe"
          : getCompilerArtifactName(process.platform, process.arch);
      const dest = join(cacheRoot, "0.6.0", destName);
      const payload = Buffer.from("fake-forst-binary");

      const fetchImpl: FetchImpl = async (url) => {
        const s = String(url);
        if (s.includes("api.github.com")) {
          return new Response(
            JSON.stringify({
              assets: [{ name: destName }],
            }),
            { status: 200, headers: { "Content-Type": "application/json" } }
          );
        }
        expect(s).toContain("/download/v0.6.0/" + destName);
        return new Response(payload, { status: 200 });
      };
      seedCompilerModuleCache(cacheRoot, "0.6.0");

      const p = await resolveForstBinary({
        version: "0.6.0",
        env: {
          ...process.env,
          FORST_CACHE_DIR: cacheRoot
        },
        fetchImpl,
        homedirFn: () => "/unused",
      });

      expect(p).toBe(dest);
      expect(readFileSync(dest).equals(payload)).toBe(true);
    } finally {
      rmSync(cacheRoot, { recursive: true, force: true });
    }
  });

  test("throws when digest missing and FORST_CLI_VERIFY=strict", async () => {
    const cacheRoot = mkdtempSync(join(tmpdir(), "forst-cli-cache-"));
    try {
      const destName =
        process.platform === "win32"
          ? "forst-windows-amd64.exe"
          : getCompilerArtifactName(process.platform, process.arch);

      const fetchImpl: FetchImpl = async (url) => {
        const s = String(url);
        if (s.includes("api.github.com")) {
          return new Response(
            JSON.stringify({
              assets: [{ name: destName }],
            }),
            { status: 200, headers: { "Content-Type": "application/json" } }
          );
        }
        return new Response(null, { status: 500 });
      };

      const err = await resolveForstBinary({
        version: "0.6.0",
        env: {
          ...process.env,
          FORST_CACHE_DIR: cacheRoot,
          FORST_CLI_VERIFY: "strict",
        },
        fetchImpl,
        homedirFn: () => "/unused",
      }).catch((e) => e);

      expect(err).toMatchObject({
        name: "CompilerBinaryDigestUnavailable",
        reason: "digest_missing",
      });
    } finally {
      rmSync(cacheRoot, { recursive: true, force: true });
    }
  });

  test("strict mode throws CompilerBinaryDigestUnavailable when release tag missing", async () => {
    const cacheRoot = mkdtempSync(join(tmpdir(), "forst-cli-cache-"));
    try {
      const fetchImpl: FetchImpl = async (url) => {
        if (String(url).includes("api.github.com")) {
          return new Response(null, { status: 404, statusText: "Not Found" });
        }
        return new Response(null, { status: 500 });
      };

      const err = await resolveForstBinary({
        version: "9.9.9",
        env: {
          ...process.env,
          FORST_CACHE_DIR: cacheRoot,
          FORST_CLI_VERIFY: "strict",
        },
        fetchImpl,
        homedirFn: () => "/unused",
      }).catch((e) => e);

      expect(err).toMatchObject({
        name: "CompilerBinaryDigestUnavailable",
        reason: "release_not_found",
      });
    } finally {
      rmSync(cacheRoot, { recursive: true, force: true });
    }
  });

  test("strict mode throws CompilerBinaryDigestUnavailable when asset missing on release", async () => {
    const cacheRoot = mkdtempSync(join(tmpdir(), "forst-cli-cache-"));
    try {
      const destName =
        process.platform === "win32"
          ? "forst-windows-amd64.exe"
          : getCompilerArtifactName(process.platform, process.arch);

      const fetchImpl: FetchImpl = async (url) => {
        if (String(url).includes("api.github.com")) {
          return new Response(
            JSON.stringify({ assets: [{ name: "other-asset" }] }),
            { status: 200, headers: { "Content-Type": "application/json" } }
          );
        }
        return new Response(null, { status: 500 });
      };

      const err = await resolveForstBinary({
        version: "0.4.0",
        env: {
          ...process.env,
          FORST_CACHE_DIR: cacheRoot,
          FORST_CLI_VERIFY: "1",
        },
        fetchImpl,
        homedirFn: () => "/unused",
      }).catch((e) => e);

      expect(err).toMatchObject({
        name: "CompilerBinaryDigestUnavailable",
        reason: "asset_not_found",
        artifact: destName,
      });
    } finally {
      rmSync(cacheRoot, { recursive: true, force: true });
    }
  });

  test("preferLatestRelease uses semver max and downloads newer release path", async () => {
    const cacheRoot = mkdtempSync(join(tmpdir(), "forst-cli-cache-"));
    try {
      const destName =
        process.platform === "win32"
          ? "forst-windows-amd64.exe"
          : getCompilerArtifactName(process.platform, process.arch);
      const dest = join(cacheRoot, "99.0.0", destName);
      const fakeBinary = Buffer.from("fake-forst-binary");
      const hex = createHash("sha256").update(fakeBinary).digest("hex");

      const fetchImpl: FetchImpl = async (url) => {
        const s = String(url);
        if (s.includes("/repos/forst-lang/forst/releases?")) {
          return new Response(
            JSON.stringify([
              { tag_name: "vscode-forst-v0.3.0" },
              { tag_name: "v99.0.0" },
            ]),
            { status: 200, headers: { "Content-Type": "application/json" } }
          );
        }
        if (s.includes("/repos/forst-lang/forst/releases/tags/v99.0.0")) {
          return new Response(
            JSON.stringify({
              assets: [{
                name: destName,
                digest: `sha256:${hex}`
              }],
            }),
            { status: 200, headers: { "Content-Type": "application/json" } }
          );
        }
        expect(s).toContain("/download/v99.0.0/" + destName);
        return new Response(fakeBinary, { status: 200 });
      };
      seedCompilerModuleCache(cacheRoot, "99.0.0");

      const p = await resolveForstBinary({
        preferLatestRelease: true,
        env: {
          ...process.env,
          FORST_CACHE_DIR: cacheRoot
        },
        fetchImpl,
        homedirFn: () => "/unused",
      });

      expect(p).toBe(dest);
      expect(readFileSync(dest).equals(fakeBinary)).toBe(true);
    } finally {
      rmSync(cacheRoot, { recursive: true, force: true });
    }
  });

  test("preferLatestRelease does not fetch when allowDownload is false", async () => {
    const cacheRoot = mkdtempSync(join(tmpdir(), "forst-cli-cache-"));
    try {
      const destName =
        process.platform === "win32"
          ? "forst-windows-amd64.exe"
          : getCompilerArtifactName(process.platform, process.arch);
      const bundled = getBundledCompilerReleaseVersion();
      const dest = join(cacheRoot, bundled, destName);
      mkdirSync(dirname(dest), { recursive: true });
      const marker = Buffer.from("cached");
      writeFileSync(dest, marker);
      if (process.platform !== "win32") {
        chmodSync(dest, 0o755);
      }
      seedCompilerModuleCache(cacheRoot, bundled);

      const fetchImpl: FetchImpl = async () => {
        throw new Error("network must not be used when allowDownload is false");
      };

      const p = await resolveForstBinary({
        preferLatestRelease: true,
        allowDownload: false,
        env: { ...process.env, ...verifyOff, FORST_CACHE_DIR: cacheRoot },
        fetchImpl,
        homedirFn: () => "/unused",
      });

      expect(p).toBe(dest);
      expect(readFileSync(dest).equals(marker)).toBe(true);
    } finally {
      rmSync(cacheRoot, { recursive: true, force: true });
    }
  });

  test("throws CompilerBinaryChecksumMismatch when digest does not match", async () => {
    const cacheRoot = mkdtempSync(join(tmpdir(), "forst-cli-cache-"));
    try {
      const destName =
        process.platform === "win32"
          ? "forst-windows-amd64.exe"
          : getCompilerArtifactName(process.platform, process.arch);
      const payload = Buffer.from("fake-forst-binary");

      const fetchImpl: FetchImpl = async (url) => {
        const s = String(url);
        if (s.includes("api.github.com")) {
          return new Response(
            JSON.stringify({
              assets: [
                {
                  name: destName,
                  digest: "sha256:" + "00".repeat(32)
                },
              ],
            }),
            { status: 200, headers: { "Content-Type": "application/json" } }
          );
        }
        return new Response(payload, { status: 200 });
      };

      const err = await resolveForstBinary({
        version: "0.6.0",
        env: {
          ...process.env,
          FORST_CACHE_DIR: cacheRoot
        },
        fetchImpl,
        homedirFn: () => "/unused",
      }).catch((e) => e);

      expect(err).toBeInstanceOf(CompilerBinaryChecksumMismatch);
      expect(String(err.message)).toContain("supply-chain");
      expect(String(err.message)).toContain("Do not run");
    } finally {
      rmSync(cacheRoot, { recursive: true, force: true });
    }
  });
});
