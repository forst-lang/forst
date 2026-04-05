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
import { resolveForstBinary } from "./resolve.js";
import {
  CompilerBinaryChecksumMismatch,
  CompilerBinaryDownloadFailed,
  CompilerBinaryDownloadHttpFailure,
} from "./errors.js";

const verifyOff = { FORST_CLI_VERIFY: "0" } as const;

test("getCompilerArtifactName matches release task naming", () => {
  expect(getCompilerArtifactName("darwin", "arm64")).toBe("forst-darwin-arm64");
  expect(getCompilerArtifactName("linux", "x64")).toBe("forst-linux-amd64");
  expect(getCompilerArtifactName("win32", "x64")).toBe("forst-windows-amd64.exe");
});

test("buildCompilerArtifactDownloadUrl", () => {
  expect(
    buildCompilerArtifactDownloadUrl("0.0.19", "forst-darwin-arm64")
  ).toBe(
    "https://github.com/forst-lang/forst/releases/download/v0.0.19/forst-darwin-arm64"
  );
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
    await expect(
      resolveForstBinary({
        env: { ...process.env, FORST_BINARY: "/nonexistent/forst-binary-xyz" },
      })
    ).rejects.toThrow(/FORST_BINARY/);
  });

  test("downloads into cache when missing", async () => {
    const cacheRoot = mkdtempSync(join(tmpdir(), "forst-cli-cache-"));
    try {
      const destName =
        process.platform === "win32"
          ? "forst-windows-amd64.exe"
          : getCompilerArtifactName(process.platform, process.arch);
      const dest = join(cacheRoot, "0.0.19", destName);

      const fakeBinary = Buffer.from("fake-forst-binary");
      const fetchImpl: typeof fetch = async (url) => {
        expect(String(url)).toContain("/download/v0.0.19/" + destName);
        return new Response(fakeBinary, { status: 200 });
      };

      const p = await resolveForstBinary({
        version: "0.0.19",
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
      const dest = join(cacheRoot, "0.0.19", destName);
      mkdirSync(dirname(dest), { recursive: true });
      const marker = Buffer.from("cached-binary-contents");
      writeFileSync(dest, marker);
      if (process.platform !== "win32") {
        chmodSync(dest, 0o755);
      }

      let fetchCalled = false;
      const fetchImpl: typeof fetch = async () => {
        fetchCalled = true;
        throw new Error("fetchImpl must not be called on cache hit");
      };

      const p = await resolveForstBinary({
        version: "0.0.19",
        env: { ...process.env, ...verifyOff, FORST_CACHE_DIR: cacheRoot },
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
    await expect(
      resolveForstBinary({
        version: "../../../evil",
        env: { ...process.env, FORST_CACHE_DIR: "/tmp" },
        homedirFn: () => "/unused",
      })
    ).rejects.toThrow(/Invalid Forst compiler version/);
  });

  test("maps HTTP failure to CompilerBinaryDownloadHttpFailure", async () => {
    const cacheRoot = mkdtempSync(join(tmpdir(), "forst-cli-cache-"));
    try {
      const fetchImpl: typeof fetch = async () =>
        new Response(null, { status: 404, statusText: "Not Found" });

      await expect(
        resolveForstBinary({
          version: "0.0.19",
          env: { ...process.env, ...verifyOff, FORST_CACHE_DIR: cacheRoot },
          fetchImpl,
          homedirFn: () => "/unused",
        })
      ).rejects.toBeInstanceOf(CompilerBinaryDownloadHttpFailure);
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
      const dest = join(cacheRoot, "0.0.19", destName);
      const payload = Buffer.from("fake-forst-binary");
      const hex = createHash("sha256").update(payload).digest("hex");

      const fetchImpl: typeof fetch = async (url) => {
        const s = String(url);
        if (s.includes("api.github.com")) {
          return new Response(
            JSON.stringify({
              assets: [{ name: destName, digest: `sha256:${hex}` }],
            }),
            { status: 200, headers: { "Content-Type": "application/json" } }
          );
        }
        expect(s).toContain("/download/v0.0.19/" + destName);
        return new Response(payload, { status: 200 });
      };

      const p = await resolveForstBinary({
        version: "0.0.19",
        env: { ...process.env, FORST_CACHE_DIR: cacheRoot },
        fetchImpl,
        homedirFn: () => "/unused",
      });

      expect(p).toBe(dest);
    } finally {
      rmSync(cacheRoot, { recursive: true, force: true });
    }
  });

  test("throws when GitHub API returns no digest and verification is required", async () => {
    const cacheRoot = mkdtempSync(join(tmpdir(), "forst-cli-cache-"));
    try {
      const destName =
        process.platform === "win32"
          ? "forst-windows-amd64.exe"
          : getCompilerArtifactName(process.platform, process.arch);

      const fetchImpl: typeof fetch = async (url) => {
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

      await expect(
        resolveForstBinary({
          version: "0.0.19",
          env: { ...process.env, FORST_CACHE_DIR: cacheRoot },
          fetchImpl,
          homedirFn: () => "/unused",
        })
      ).rejects.toBeInstanceOf(CompilerBinaryDownloadFailed);
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

      const fetchImpl: typeof fetch = async (url) => {
        const s = String(url);
        if (s.includes("api.github.com")) {
          return new Response(
            JSON.stringify({
              assets: [
                { name: destName, digest: "sha256:" + "00".repeat(32) },
              ],
            }),
            { status: 200, headers: { "Content-Type": "application/json" } }
          );
        }
        return new Response(payload, { status: 200 });
      };

      await expect(
        resolveForstBinary({
          version: "0.0.19",
          env: { ...process.env, FORST_CACHE_DIR: cacheRoot },
          fetchImpl,
          homedirFn: () => "/unused",
        })
      ).rejects.toBeInstanceOf(CompilerBinaryChecksumMismatch);
    } finally {
      rmSync(cacheRoot, { recursive: true, force: true });
    }
  });
});
