import { describe, expect, test } from "bun:test";
import {
  existsSync,
  mkdtempSync,
  mkdirSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import {
  needsOfficialPluginBundle,
  parseGenerateArgv,
  shouldDownloadPluginsForSpawn,
} from "./ftconfig-plugins.js";

const fs = { existsSync, readFileSync };

describe("parseGenerateArgv", () => {
  test("returns null for non-generate commands", () => {
    expect(parseGenerateArgv(["version"])).toBeNull();
  });

  test("parses generate target and flags", () => {
    expect(
      parseGenerateArgv([
        "generate",
        "--dump-semantic",
        "--config",
        "cfg.json",
        "./app",
      ])
    ).toEqual({
      target: "./app",
      dumpSemantic: true,
      listManifest: false,
      configPath: "cfg.json",
    });
  });
});

describe("needsOfficialPluginBundle", () => {
  test("true for bare official cmd", () => {
    expect(needsOfficialPluginBundle(["forst-gen-echo"])).toBe(true);
  });

  test("false for project-local relative cmd", () => {
    expect(needsOfficialPluginBundle(["./bin/forst-gen-echo"])).toBe(false);
  });
});

describe("shouldDownloadPluginsForSpawn", () => {
  test("false when ftconfig has no plugins", () => {
    const root = mkdtempSync(join(tmpdir(), "forst-ftconfig-plugins-"));
    try {
      writeFileSync(
        join(root, "ftconfig.json"),
        JSON.stringify({ generate: { plugins: [] } }),
        "utf8"
      );
      expect(
        shouldDownloadPluginsForSpawn({
          argv: ["generate", "."],
          cwd: root,
          fs,
        })
      ).toBe(false);
    } finally {
      rmSync(root, { recursive: true, force: true });
    }
  });

  test("true when generate references official plugin", () => {
    const root = mkdtempSync(join(tmpdir(), "forst-ftconfig-plugins-"));
    try {
      writeFileSync(
        join(root, "ftconfig.json"),
        JSON.stringify({
          generate: {
            plugins: [{ name: "echo", cmd: "forst-gen-echo", out: "generated/echo" }],
          },
        }),
        "utf8"
      );
      expect(
        shouldDownloadPluginsForSpawn({
          argv: ["generate", "."],
          cwd: root,
          fs,
        })
      ).toBe(true);
    } finally {
      rmSync(root, { recursive: true, force: true });
    }
  });

  test("false for dump-semantic even with plugins configured", () => {
    const root = mkdtempSync(join(tmpdir(), "forst-ftconfig-plugins-"));
    try {
      writeFileSync(
        join(root, "ftconfig.json"),
        JSON.stringify({
          generate: {
            plugins: [{ name: "echo", cmd: "forst-gen-echo", out: "generated/echo" }],
          },
        }),
        "utf8"
      );
      expect(
        shouldDownloadPluginsForSpawn({
          argv: ["generate", "--dump-semantic", "."],
          cwd: root,
          fs,
        })
      ).toBe(false);
    } finally {
      rmSync(root, { recursive: true, force: true });
    }
  });

  test("finds ftconfig in ancestor of nested target", () => {
    const root = mkdtempSync(join(tmpdir(), "forst-ftconfig-plugins-"));
    try {
      const nested = join(root, "pkg", "api");
      mkdirSync(nested, { recursive: true });
      writeFileSync(
        join(root, "ftconfig.json"),
        JSON.stringify({
          generate: {
            plugins: [{ name: "echo", cmd: "forst-gen-echo", out: "generated/echo" }],
          },
        }),
        "utf8"
      );
      expect(
        shouldDownloadPluginsForSpawn({
          argv: ["generate", "pkg/api"],
          cwd: root,
          fs,
        })
      ).toBe(true);
    } finally {
      rmSync(root, { recursive: true, force: true });
    }
  });
});
