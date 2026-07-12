import path from "node:path";
import { describe, expect, it } from "bun:test";
import { parseForstIndexModuleV1 } from "../manifest/schema.js";
import { parseCliArgs, runCli } from "./cli.js";
import { emitForstIndexV1 } from "./emit-forst-index-v1.js";

const repoRoot = path.resolve(import.meta.dir, "../../../..");
const boundaryRoot = path.join(repoRoot, "examples/in/rfc/node-interop");
const paymentFixture = path.join(boundaryRoot, "legacy/payment.ts");
const fixtureRoot = path.join(import.meta.dir, "../../test/fixtures");

describe("forst-node-index CLI", () => {
  it("parses --root, --format, and --files", () => {
    const options = parseCliArgs([
      "--root",
      "/tmp/project",
      "--format",
      "forst-index-v1",
      "--files",
      "legacy/payment.ts,legacy/events.ts",
    ]);

    expect(options).toEqual({
      root: "/tmp/project",
      format: "forst-index-v1",
      files: ["legacy/payment.ts", "legacy/events.ts"],
    });
  });

  it("indexes payment.ts exports from the node-interop example", () => {
    const index = emitForstIndexV1({
      root: boundaryRoot,
      files: ["legacy/payment.ts"],
    });

    expect(index.format).toBe("forst-index-v1");
    expect(index.modules).toHaveLength(1);

    const module = index.modules[0];
    expect(module.moduleId).toBe("legacy/payment.ts");

    const createExport = module.exports.find((entry) => entry.name === "create");
    expect(createExport).toMatchObject({
      name: "create",
      kind: "function",
      parameters: [
        { name: "amount", type: { kind: "number" } },
        { name: "currency", type: { kind: "string" } },
      ],
      returnType: {
        kind: "object",
        fields: {
          id: { kind: "string" },
        },
      },
      definition: {
        line: 1,
        column: 17,
        endLine: 1,
        endColumn: 23,
      },
    });

    expect(parseForstIndexModuleV1(module).exports[0]?.name).toBe("create");
  });

  it("indexes export const arrow functions as function kind", () => {
    const index = emitForstIndexV1({
      root: fixtureRoot,
      files: ["arrow-export.ts"],
    });

    const module = index.modules[0];
    expect(module.moduleId).toBe("arrow-export.ts");
    expect(module.exports).toHaveLength(1);
    expect(module.exports[0]).toMatchObject({
      name: "create",
      kind: "function",
      parameters: [
        { name: "amount", type: { kind: "number" } },
        { name: "currency", type: { kind: "string" } },
      ],
      returnType: {
        kind: "object",
        fields: { id: { kind: "string" } },
      },
    });
  });

  it("indexes re-exports from other modules", () => {
    const index = emitForstIndexV1({
      root: fixtureRoot,
      files: ["re-export-barrel.ts"],
    });

    const module = index.modules[0];
    expect(module.moduleId).toBe("re-export-barrel.ts");
    expect(module.exports).toHaveLength(1);
    expect(module.exports[0]).toMatchObject({
      name: "greet",
      kind: "function",
      parameters: [{ name: "name", type: { kind: "string" } }],
      returnType: { kind: "string" },
      definition: {
        file: "re-export-source.ts",
        line: 1,
        column: 17,
        endLine: 1,
        endColumn: 22,
      },
    });
  });

  it("indexes async functions, generators, and async generators", () => {
    const index = emitForstIndexV1({
      root: fixtureRoot,
      files: ["payment-like.ts"],
    });

    const module = index.modules[0];
    expect(module.exports.map((entry) => entry.name)).toEqual([
      "create",
      "formatAmount",
      "readChunks",
      "watchEvents",
    ]);

    expect(module.exports.find((entry) => entry.name === "create")).toMatchObject({
      kind: "asyncFunction",
      returnType: {
        kind: "object",
        fields: { id: { kind: "string" } },
      },
    });

    expect(module.exports.find((entry) => entry.name === "watchEvents")).toMatchObject({
      kind: "asyncGenerator",
      yieldType: {
        kind: "object",
        fields: { type: { kind: "string" } },
      },
      returnType: { kind: "void" },
    });

    expect(module.exports.find((entry) => entry.name === "readChunks")).toMatchObject({
      kind: "generator",
      yieldType: {
        $binary: true,
        element: { kind: "bytes" },
      },
    });
  });

  it("runs CLI against the payment fixture and prints JSON", () => {
    const stdout = Bun.spawnSync({
      cmd: [
        process.execPath,
        path.join(import.meta.dir, "cli.ts"),
        "--root",
        boundaryRoot,
        "--format",
        "forst-index-v1",
        "--files",
        "legacy/payment.ts",
      ],
      stdout: "pipe",
      stderr: "pipe",
    });

    expect(stdout.exitCode).toBe(0);
    const parsed = JSON.parse(stdout.stdout.toString());
    expect(parsed.format).toBe("forst-index-v1");
    expect(parsed.modules[0].moduleId).toBe("legacy/payment.ts");
    expect(
      parsed.modules[0].exports.some((entry: { name: string }) => entry.name === "create"),
    ).toBe(true);
  });

  it("returns non-zero when format is unsupported", () => {
    const code = runCli([
      "--root",
      boundaryRoot,
      "--format",
      "unknown",
      "--files",
      "legacy/payment.ts",
    ]);
    expect(code).toBe(1);
  });
});

describe("payment fixture exists", () => {
  it("is present at examples/in/rfc/node-interop/legacy/payment.ts", async () => {
    const file = Bun.file(paymentFixture);
    expect(await file.exists()).toBe(true);
  });
});
