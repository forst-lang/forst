import { afterEach, describe, expect, it } from "bun:test";
import { mergeForstSidecarEnv } from "./config-merge";

describe("mergeForstSidecarEnv", () => {
  const env = { ...process.env };

  afterEach(() => {
    process.env = { ...env };
  });

  it("defaults versionCheck to warn", () => {
    const m = mergeForstSidecarEnv({});
    expect(m.versionCheck).toBe("warn");
  });

  it("defaults to spawn when no env hints", () => {
    delete process.env.FORST_DEV_URL;
    delete process.env.FORST_SKIP_SPAWN;
    const m = mergeForstSidecarEnv({});
    expect(m.sidecarRuntime).toBe("spawn");
  });

  it("uses connect when FORST_DEV_URL is set and runtime not overridden", () => {
    process.env.FORST_DEV_URL = "http://127.0.0.1:7777";
    delete process.env.FORST_SKIP_SPAWN;
    const m = mergeForstSidecarEnv({});
    expect(m.sidecarRuntime).toBe("connect");
    expect(m.devServerUrl).toBe("http://127.0.0.1:7777");
  });

  it("uses connect when FORST_SKIP_SPAWN=1", () => {
    delete process.env.FORST_DEV_URL;
    process.env.FORST_SKIP_SPAWN = "1";
    const m = mergeForstSidecarEnv({});
    expect(m.sidecarRuntime).toBe("connect");
  });

  it("respects explicit sidecarRuntime spawn even if FORST_DEV_URL is set", () => {
    process.env.FORST_DEV_URL = "http://127.0.0.1:7777";
    const m = mergeForstSidecarEnv({ sidecarRuntime: "spawn" });
    expect(m.sidecarRuntime).toBe("spawn");
  });

  it("applies FORST_PORT when port omitted", () => {
    process.env.FORST_PORT = "3001";
    const m = mergeForstSidecarEnv({});
    expect(m.port).toBe(3001);
  });

  it("applies FORST_DIR when forstDir omitted", () => {
    process.env.FORST_DIR = "./packages/api/forst";
    const m = mergeForstSidecarEnv({});
    expect(m.forstDir).toBe("./packages/api/forst");
  });

  it("defaults downloadCompiler to false when unset", () => {
    delete process.env.FORST_DOWNLOAD_COMPILER;
    const m = mergeForstSidecarEnv({});
    expect(m.downloadCompiler).toBe(false);
  });

  it("enables downloadCompiler when FORST_DOWNLOAD_COMPILER=1", () => {
    process.env.FORST_DOWNLOAD_COMPILER = "1";
    const m = mergeForstSidecarEnv({});
    expect(m.downloadCompiler).toBe(true);
  });

  it("respects explicit downloadCompiler over env", () => {
    process.env.FORST_DOWNLOAD_COMPILER = "1";
    const m = mergeForstSidecarEnv({ downloadCompiler: false });
    expect(m.downloadCompiler).toBe(false);
  });
});
