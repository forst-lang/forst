import { describe, expect, test } from "bun:test";
import { mkdtempSync, readFileSync, rmSync, writeFileSync, chmodSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { getCompilerArtifactName } from "./artifact.js";
import { buildCompilerArtifactDownloadUrl } from "./urls.js";
import { resolveForstBinary } from "./resolve.js";
import { CompilerBinaryDownloadHttpFailure } from "./errors.js";

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
        env: { ...process.env, FORST_CACHE_DIR: cacheRoot },
        fetchImpl,
        homedirFn: () => "/unused",
      });

      expect(p).toBe(dest);
      expect(readFileSync(dest).equals(fakeBinary)).toBe(true);
    } finally {
      rmSync(cacheRoot, { recursive: true, force: true });
    }
  });

  test("maps HTTP failure to CompilerBinaryDownloadHttpFailure", async () => {
    const cacheRoot = mkdtempSync(join(tmpdir(), "forst-cli-cache-"));
    try {
      const fetchImpl: typeof fetch = async () =>
        new Response(null, { status: 404, statusText: "Not Found" });

      await expect(
        resolveForstBinary({
          version: "0.0.19",
          env: { ...process.env, FORST_CACHE_DIR: cacheRoot },
          fetchImpl,
          homedirFn: () => "/unused",
        })
      ).rejects.toBeInstanceOf(CompilerBinaryDownloadHttpFailure);
    } finally {
      rmSync(cacheRoot, { recursive: true, force: true });
    }
  });
});
