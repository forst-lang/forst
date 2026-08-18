import { describe, expect, test } from "bun:test";
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { getPluginsArtifactName } from "./artifact.js";
import {
  getPluginsDirForVersion,
  pluginsReady,
  prependPluginDirsToPath,
} from "./plugins-module.js";

describe("getPluginsArtifactName", () => {
  test("darwin arm64 tarball", () => {
    expect(getPluginsArtifactName("darwin", "arm64")).toBe(
      "forst-plugins-darwin-arm64.tar.gz"
    );
  });

  test("windows amd64 zip", () => {
    expect(getPluginsArtifactName("win32", "x64")).toBe(
      "forst-plugins-windows-amd64.zip"
    );
  });
});

describe("pluginsReady", () => {
  test("detects official plugin binary", () => {
    const dir = mkdtempSync(join(tmpdir(), "forst-plugins-ready-"));
    try {
      writeFileSync(join(dir, "forst-gen-echo"), "bin", { mode: 0o755 });
      expect(pluginsReady(dir)).toBe(true);
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  });
});

describe("prependPluginDirsToPath", () => {
  test("prepends directories", () => {
    const env = prependPluginDirsToPath({ PATH: "/usr/bin" }, "/cache/0.1.0");
    expect(env.PATH?.startsWith("/cache/0.1.0")).toBe(true);
    expect(env.PATH).toContain("/usr/bin");
  });
});

describe("getPluginsDirForVersion", () => {
  test("matches compiler cache layout", () => {
    const root = mkdtempSync(join(tmpdir(), "forst-plugins-cache-"));
    try {
      const dir = getPluginsDirForVersion("1.2.3", {
        env: { FORST_CACHE_DIR: root },
      });
      expect(dir).toBe(join(root, "1.2.3"));
    } finally {
      rmSync(root, { recursive: true, force: true });
    }
  });
});
