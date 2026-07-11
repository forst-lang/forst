import { describe, expect, it } from "bun:test";
import {
  FORST_INDEX_V1_FORMAT,
  FORST_NODE_MANIFEST_V1_VERSION,
  ForstNodeSchemaValidationError,
  isForstNodeExportKind,
  isValidModuleId,
  parseForstIndexModuleV1,
  parseForstNodeManifestV1,
} from "./schema.js";

describe("forst-node-manifest-v1", () => {
  it("parses a valid manifest", () => {
    const manifest = parseForstNodeManifestV1({
      version: 1,
      boundaryRoot: "/abs/path/to/project",
      exports: [
        {
          moduleId: "legacy/payment.ts",
          name: "create",
          kind: "asyncFunction",
        },
        {
          moduleId: "legacy/payment.ts",
          name: "watchEvents",
          kind: "asyncGenerator",
        },
      ],
    });

    expect(manifest.version).toBe(FORST_NODE_MANIFEST_V1_VERSION);
    expect(manifest.boundaryRoot).toBe("/abs/path/to/project");
    expect(manifest.exports).toHaveLength(2);
    expect(manifest.exports[0]?.kind).toBe("asyncFunction");
  });

  it("rejects unsupported manifest version", () => {
    expect(() =>
      parseForstNodeManifestV1({
        version: 2,
        boundaryRoot: "/project",
        exports: [],
      }),
    ).toThrow(ForstNodeSchemaValidationError);
  });

  it("rejects invalid export kind", () => {
    expect(() =>
      parseForstNodeManifestV1({
        version: 1,
        boundaryRoot: "/project",
        exports: [{ moduleId: "a.ts", name: "fn", kind: "method" }],
      }),
    ).toThrow(ForstNodeSchemaValidationError);
  });

  it("rejects moduleId with parent traversal", () => {
    expect(isValidModuleId("../escape.ts")).toBe(false);
    expect(() =>
      parseForstNodeManifestV1({
        version: 1,
        boundaryRoot: "/project",
        exports: [{ moduleId: "../escape.ts", name: "fn", kind: "function" }],
      }),
    ).toThrow(ForstNodeSchemaValidationError);
  });

  it("accepts all export kinds", () => {
    for (const kind of [
      "function",
      "asyncFunction",
      "generator",
      "asyncGenerator",
    ] as const) {
      expect(isForstNodeExportKind(kind)).toBe(true);
      const manifest = parseForstNodeManifestV1({
        version: 1,
        boundaryRoot: "/project",
        exports: [{ moduleId: "legacy/payment.ts", name: "fn", kind }],
      });
      expect(manifest.exports[0]?.kind).toBe(kind);
    }
  });
});

describe("forst-index-v1", () => {
  it("parses a valid module index", () => {
    const index = parseForstIndexModuleV1({
      moduleId: "legacy/payment.ts",
      exports: [
        {
          name: "create",
          kind: "asyncFunction",
          parameters: [
            { name: "amount", type: { kind: "number" } },
            { name: "currency", type: { kind: "string" } },
          ],
          returnType: {
            kind: "object",
            fields: { id: { kind: "string" } },
          },
        },
        {
          name: "readChunks",
          kind: "generator",
          parameters: [{ name: "path", type: { kind: "string" } }],
          yieldType: {
            $binary: true,
            element: { kind: "bytes" },
          },
        },
      ],
    });

    expect(FORST_INDEX_V1_FORMAT).toBe("forst-index-v1");
    expect(index.moduleId).toBe("legacy/payment.ts");
    expect(index.exports[0]?.returnType?.fields?.id?.kind).toBe("string");
    expect(index.exports[1]?.yieldType?.$binary).toBe(true);
  });

  it("rejects invalid moduleId in index", () => {
    expect(() =>
      parseForstIndexModuleV1({
        moduleId: "/abs/payment.ts",
        exports: [],
      }),
    ).toThrow(ForstNodeSchemaValidationError);
  });
});
