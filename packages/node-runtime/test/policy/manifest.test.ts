import { describe, expect, test } from "bun:test";
import path from "node:path";
import { fileURLToPath } from "node:url";
import {
  assertExportAllowed,
  buildManifestIndex,
  validateManifest,
} from "../../src/policy/manifest.js";
import type { ForstNodeManifestExportV1 } from "../../src/manifest/schema.js";
import { validateModuleIdSyntax } from "../../src/policy/paths.js";
import { FORBIDDEN } from "../../src/rpc/errors.js";

const testDir = path.dirname(fileURLToPath(import.meta.url));
const fixtureRoot = path.resolve(testDir, "..");
const syncModuleId = "fixtures/sync-add.ts";

function sampleManifest(boundaryRoot: string) {
  return {
    version: 1 as const,
    boundaryRoot,
    exports: [
      {
        moduleId: syncModuleId,
        name: "add",
        kind: "function" as const,
      },
      {
        moduleId: syncModuleId,
        name: "greet",
        kind: "function" as const,
      },
    ],
  };
}

describe("validateManifest", () => {
  test("accepts forst-node-manifest-v1 shape", () => {
    const manifest = validateManifest(sampleManifest(fixtureRoot));
    expect(manifest.exports).toHaveLength(2);
  });

  test("rejects unsupported manifest version", () => {
    expect(() =>
      validateManifest({ ...sampleManifest(fixtureRoot), version: 2 })
    ).toThrow();
  });
});

describe("assertExportAllowed", () => {
  test("allows manifest-listed export", () => {
    const index = buildManifestIndex(sampleManifest(fixtureRoot));
    const entry = assertExportAllowed(index, syncModuleId, "add", "function");
    expect(entry.name).toBe("add");
  });

  test("rejects unknown export with forbidden code", () => {
    const index = buildManifestIndex(sampleManifest(fixtureRoot));
    try {
      assertExportAllowed(index, syncModuleId, "missing", "function");
      expect.unreachable();
    } catch (err) {
      expect(err).toMatchObject({ code: FORBIDDEN });
    }
  });

  test("rejects kind mismatch for sync call", () => {
    const manifest = sampleManifest(fixtureRoot);
    (manifest.exports as ForstNodeManifestExportV1[])[0] = {
      moduleId: syncModuleId,
      name: "add",
      kind: "asyncFunction",
    };
    const index = buildManifestIndex(manifest);
    try {
      assertExportAllowed(index, syncModuleId, "add", "function");
      expect.unreachable();
    } catch (err) {
      expect(err).toMatchObject({ code: FORBIDDEN });
    }
  });
});

describe("validateModuleIdSyntax", () => {
  test("rejects parent traversal", () => {
    try {
      validateModuleIdSyntax("../secret.ts");
      expect.unreachable();
    } catch (err) {
      expect(err).toMatchObject({ code: FORBIDDEN });
    }
  });

  test("rejects absolute paths", () => {
    try {
      validateModuleIdSyntax("/etc/passwd.ts");
      expect.unreachable();
    } catch (err) {
      expect(err).toMatchObject({ code: FORBIDDEN });
    }
  });

  test("rejects file:// URLs", () => {
    try {
      validateModuleIdSyntax("file:///tmp/x.ts");
      expect.unreachable();
    } catch (err) {
      expect(err).toMatchObject({ code: FORBIDDEN });
    }
  });
});
