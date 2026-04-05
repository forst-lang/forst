import { describe, expect, it } from "bun:test";
import {
  buildForstDevSpawnArgs,
  buildForstGenerateArgs,
  buildForstWatchRoots,
  effectiveProjectRootDir,
  effectiveWatchDirForConfig,
} from "./server";
import type { ForstConfig } from "./types";

describe("buildForstDevSpawnArgs", () => {
  it("includes -config with an absolute path when configPath is set", () => {
    const cfg: ForstConfig = {
      rootDir: "./repo",
      configPath: "./cfg/ftconfig.json",
      logLevel: "warn",
    };
    const { args, cwd } = buildForstDevSpawnArgs(cfg, 9090);
    expect(args[0]).toBe("dev");
    expect(args).toContain("-port");
    expect(args).toContain("9090");
    expect(args).toContain("-root");
    expect(args).toContain(cwd);
    expect(args).toContain("-config");
    const configIdx = args.indexOf("-config");
    expect(configIdx).toBeGreaterThan(-1);
    expect(args[configIdx + 1]).toMatch(/ftconfig\.json$/);
    expect(args).toContain("-log-level");
    expect(args).toContain("warn");
  });

  it("omits -config when configPath is absent", () => {
    const cfg: ForstConfig = { forstDir: "./forst" };
    const { args } = buildForstDevSpawnArgs(cfg, 8080);
    expect(args).not.toContain("-config");
  });
});

describe("buildForstGenerateArgs", () => {
  it("includes -config when configPath is set", () => {
    const cfg: ForstConfig = { configPath: "./cfg/ftconfig.json" };
    const args = buildForstGenerateArgs(cfg, "/abs/root");
    expect(args[0]).toBe("generate");
    expect(args).toContain("-config");
    const i = args.indexOf("-config");
    expect(args[i + 1]).toMatch(/ftconfig\.json$/);
    expect(args[args.length - 1]).toBe("/abs/root");
  });

  it("omits -config when configPath is absent", () => {
    const cfg: ForstConfig = {};
    const args = buildForstGenerateArgs(cfg, "/abs/root");
    expect(args).toEqual(["generate", "/abs/root"]);
  });
});

describe("buildForstWatchRoots", () => {
  it("uses watchRoots when provided and paths exist", () => {
    const cfg: ForstConfig = {
      watchRoots: ["./src"],
    };
    const roots = buildForstWatchRoots(cfg);
    expect(Array.isArray(roots)).toBe(true);
  });
});

describe("effectiveProjectRootDir / effectiveWatchDirForConfig", () => {
  it("prefers rootDir for project root", () => {
    const cfg: ForstConfig = {
      rootDir: "/tmp/monorepo",
      forstDir: "/tmp/monorepo/forst",
    };
    expect(effectiveProjectRootDir(cfg)).toBe("/tmp/monorepo");
  });

  it("prefers forstDir for default watch when both set", () => {
    const cfg: ForstConfig = {
      rootDir: "/tmp/monorepo",
      forstDir: "/tmp/monorepo/pkg/forst",
    };
    expect(effectiveWatchDirForConfig(cfg)).toBe("/tmp/monorepo/pkg/forst");
  });
});
