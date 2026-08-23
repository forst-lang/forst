import { describe, expect, test } from "bun:test";
import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import {
  matchesExcludePatterns,
  matchesGlobPattern,
  resolveModulePath,
  setFilesExcludePatterns,
  validateModuleIdSyntax,
} from "../../src/policy/paths.js";

describe("paths exclude patterns", () => {
  test("matchesGlobPattern supports doublestar segments", () => {
    expect(matchesGlobPattern("**/node_modules/**", "pkg/node_modules/x.ts")).toBe(
      true
    );
    expect(matchesGlobPattern("**/*.skip.ts", "legacy/payment.skip.ts")).toBe(
      true
    );
    expect(matchesGlobPattern("**/*.skip.ts", "legacy/payment.ts")).toBe(false);
  });

  test("validateModuleIdSyntax rejects files.exclude matches from initialize params", () => {
    setFilesExcludePatterns(["**/*.skip.ts", "**/secret/**"]);
    expect(() => validateModuleIdSyntax("legacy/payment.skip.ts")).toThrow(
      /files\.exclude/
    );
    expect(() => validateModuleIdSyntax("secret/payment.ts")).toThrow(
      /files\.exclude/
    );
    expect(() => validateModuleIdSyntax("legacy/payment.ts")).not.toThrow();
  });

  test("validateModuleIdSyntax rejects .forst paths under files.exclude", () => {
    setFilesExcludePatterns(["**/.forst/**"]);
    expect(() =>
      validateModuleIdSyntax("legacy/payment.js")
    ).not.toThrow();
    expect(() =>
      validateModuleIdSyntax(".forst/client/main.js")
    ).toThrow(/files\.exclude/);
  });

  test("matchesExcludePatterns returns false when patterns empty", () => {
    expect(matchesExcludePatterns("legacy/payment.ts", [])).toBe(false);
  });
});

describe("resolveModulePath", () => {
  test("accepts fixture file under boundaryRoot", async () => {
    const testDir = path.dirname(fileURLToPath(import.meta.url));
    const boundaryRoot = path.resolve(testDir, "..");
    const abs = await resolveModulePath(boundaryRoot, "fixtures/sync-add.ts");
    expect(abs.endsWith("fixtures/sync-add.ts")).toBe(true);
  });

  test("rejects missing file", async () => {
    const boundaryRoot = await fs.mkdtemp(path.join(os.tmpdir(), "forst-path-"));
    try {
      expect(
        resolveModulePath(boundaryRoot, "legacy/missing.ts")
      ).rejects.toThrow(/does not resolve/);
    } finally {
      await fs.rm(boundaryRoot, { recursive: true, force: true });
    }
  });

  test("rejects directory target", async () => {
    const boundaryRoot = await fs.mkdtemp(path.join(os.tmpdir(), "forst-path-"));
    try {
      await fs.mkdir(path.join(boundaryRoot, "legacy.ts"), { recursive: true });
      expect(
        resolveModulePath(boundaryRoot, "legacy.ts")
      ).rejects.toThrow(/must refer to a regular file/);
    } finally {
      await fs.rm(boundaryRoot, { recursive: true, force: true });
    }
  });

  test("rejects symlink escape outside boundaryRoot", async () => {
    const outside = await fs.mkdtemp(path.join(os.tmpdir(), "forst-out-"));
    const boundaryRoot = await fs.mkdtemp(path.join(os.tmpdir(), "forst-bound-"));
    try {
      const outsideFile = path.join(outside, "escape.ts");
      await fs.writeFile(outsideFile, "export const x = 1;\n");

      const linkDir = path.join(boundaryRoot, "legacy");
      await fs.mkdir(linkDir, { recursive: true });
      const linkPath = path.join(linkDir, "escape.ts");
      await fs.symlink(outsideFile, linkPath);

      expect(
        resolveModulePath(boundaryRoot, "legacy/escape.ts")
      ).rejects.toThrow(/escapes boundaryRoot/);
    } finally {
      await fs.rm(boundaryRoot, { recursive: true, force: true });
      await fs.rm(outside, { recursive: true, force: true });
    }
  });

  test("resolves compiled .js under modulesDir when set", async () => {
    const boundaryRoot = await fs.mkdtemp(path.join(os.tmpdir(), "forst-bound-"));
    const modulesDir = await fs.mkdtemp(path.join(os.tmpdir(), "forst-modules-"));
    try {
      await fs.mkdir(path.join(modulesDir, "legacy"), { recursive: true });
      const modulePath = path.join(modulesDir, "legacy", "payment.js");
      await fs.writeFile(modulePath, "export function add() {}\n");

      const abs = await resolveModulePath(
        boundaryRoot,
        "legacy/payment.js",
        modulesDir
      );
      expect(abs.endsWith("legacy/payment.js")).toBe(true);
    } finally {
      await fs.rm(boundaryRoot, { recursive: true, force: true });
      await fs.rm(modulesDir, { recursive: true, force: true });
    }
  });

  test("rejects symlink escape outside modulesDir", async () => {
    const outside = await fs.mkdtemp(path.join(os.tmpdir(), "forst-out-"));
    const boundaryRoot = await fs.mkdtemp(path.join(os.tmpdir(), "forst-bound-"));
    const modulesDir = await fs.mkdtemp(path.join(os.tmpdir(), "forst-modules-"));
    try {
      const outsideFile = path.join(outside, "escape.js");
      await fs.writeFile(outsideFile, "export const x = 1;\n");

      const linkDir = path.join(modulesDir, "legacy");
      await fs.mkdir(linkDir, { recursive: true });
      const linkPath = path.join(linkDir, "escape.js");
      await fs.symlink(outsideFile, linkPath);

      expect(
        resolveModulePath(boundaryRoot, "legacy/escape.js", modulesDir)
      ).rejects.toThrow(/compiled modules directory/);
    } finally {
      await fs.rm(boundaryRoot, { recursive: true, force: true });
      await fs.rm(modulesDir, { recursive: true, force: true });
      await fs.rm(outside, { recursive: true, force: true });
    }
  });
});
