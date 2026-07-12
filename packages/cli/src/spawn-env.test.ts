import { describe, expect, test } from "bun:test";
import { existsSync, mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { buildForstSpawnEnv } from "./spawn-env.js";

describe("buildForstSpawnEnv", () => {
  test("sets FORST_GOMOD_ROOT to cached module when present", async () => {
    const cacheRoot = join("/tmp", "forst-spawn-env-test-" + Date.now());
    const version = "9.9.9";
    const versionDir = join(cacheRoot, version);
    const moduleDir = join(versionDir, "module");
    mkdirSync(join(moduleDir, "cmd", "forst"), { recursive: true });
    writeFileSync(join(moduleDir, "go.mod"), "module forst\n");

    const binaryPath = join(versionDir, "forst-darwin-arm64");
    writeFileSync(binaryPath, "fake-binary");

    const env: NodeJS.ProcessEnv = {
      FORST_CACHE_DIR: cacheRoot,
      PATH: "/usr/bin",
    };

    const fetchFn = async () =>
      new Response(new Uint8Array(), { status: 404 });

    const { bin, env: spawnEnv } = await buildForstSpawnEnv({
      version,
      allowDownload: false,
      env,
      fetchFn,
      fs: {
        existsSync,
        mkdirSync,
        readFileSync: () => {
          throw new Error("unexpected read");
        },
        writeFileSync,
        chmodSync: () => {},
        renameSync: () => {},
        unlinkSync: () => {},
        statSync: () => ({ mtimeMs: Date.now() } as never),
      },
    });

    expect(bin).toBe(binaryPath);
    expect(spawnEnv.FORST_GOMOD_ROOT).toBe(moduleDir);
  });

  test("does not override user FORST_GOMOD_ROOT", async () => {
    const userRoot = "/custom/forst/module";
    const env: NodeJS.ProcessEnv = {
      FORST_BINARY: "/bin/forst",
      FORST_GOMOD_ROOT: userRoot,
    };
    const fs = {
      existsSync: (p: string) => p === "/bin/forst",
      mkdirSync: () => {},
      readFileSync: () => {
        throw new Error("unexpected");
      },
      writeFileSync: () => {},
      chmodSync: () => {},
      renameSync: () => {},
      unlinkSync: () => {},
      statSync: () => ({ mtimeMs: Date.now() } as never),
    };
    const { env: spawnEnv } = await buildForstSpawnEnv({ env, fs });
    expect(spawnEnv.FORST_GOMOD_ROOT).toBe(userRoot);
  });
});
