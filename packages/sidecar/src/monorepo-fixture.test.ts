import { join } from "node:path";
import { describe, expect, it } from "bun:test";
import {
  buildForstDevSpawnArgs,
  buildForstWatchRoots,
  effectiveProjectRootDir,
} from "./server";
import type { ForstConfig } from "./types";

const monorepoRoot = join(import.meta.dir, "..", "testdata", "monorepo");

describe("testdata/monorepo fixture", () => {
  it("resolves multi-package watch roots", () => {
    const cfg: ForstConfig = {
      rootDir: monorepoRoot,
      watchRoots: [
        join(monorepoRoot, "apps", "api"),
        join(monorepoRoot, "packages", "lib"),
      ],
    };
    const roots = buildForstWatchRoots(cfg);
    expect(roots).toHaveLength(2);
    expect(roots[0]).toContain("apps");
    expect(roots[1]).toContain("packages");
  });

  it("uses monorepo root for forst dev -root and -config", () => {
    const cfg: ForstConfig = {
      rootDir: monorepoRoot,
      configPath: join(monorepoRoot, "ftconfig.json"),
    };
    const { args, cwd } = buildForstDevSpawnArgs(cfg, 8081);
    expect(effectiveProjectRootDir(cfg)).toBe(monorepoRoot);
    expect(cwd).toBe(monorepoRoot);
    expect(args).toContain("-config");
    expect(args[args.indexOf("-config") + 1]).toContain("ftconfig.json");
  });
});
