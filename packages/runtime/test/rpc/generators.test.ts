import { describe, expect, test } from "bun:test";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { createDispatcher } from "../../src/rpc/dispatcher.js";
import * as Errors from "../../src/rpc/errors.js";
import {
  METHOD_GEN_CLOSE,
  METHOD_GEN_NEXT,
  METHOD_GEN_NEXT_BATCH,
  METHOD_GEN_OPEN,
  METHOD_INITIALIZE,
  PROTOCOL_VERSION,
  WIRE_PROTOCOL_PROTO_V1,
} from "../../src/rpc/protocol.js";
import { clearModuleCache } from "../../src/runtime/module_cache.js";
import {
  openStreamCount,
  resetGeneratorStateForTest,
} from "../../src/runtime/generators.js";
import { runTestEffect } from "../helpers/run-effect.js";

const testDir = path.dirname(fileURLToPath(import.meta.url));
const fixtureRoot = path.resolve(testDir, "..");
const moduleId = "fixtures/generators.ts";

function manifest(boundaryRoot: string) {
  return {
    version: 1 as const,
    boundaryRoot,
    exports: [
      { moduleId, name: "syncNumbers", kind: "generator" as const },
      { moduleId, name: "asyncNumbers", kind: "asyncGenerator" as const },
      { moduleId, name: "emptyGen", kind: "generator" as const },
      { moduleId, name: "throwGen", kind: "generator" as const },
      { moduleId, name: "withFinally", kind: "generator" as const },
    ],
  };
}

function initializeParams(boundaryRoot: string) {
  return {
    protocolVersion: PROTOCOL_VERSION,
    boundaryRoot,
    manifest: manifest(boundaryRoot),
    supportedProtocols: [WIRE_PROTOCOL_PROTO_V1],
  };
}

describe("generator RPC", () => {
  test("sync generator yields ordered values then done", async () => {
    clearModuleCache();
    resetGeneratorStateForTest();
    const { dispatch } = createDispatcher();
    await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 1,
      method: METHOD_INITIALIZE,
      params: initializeParams(fixtureRoot),
    }));

    const open = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 2,
      method: METHOD_GEN_OPEN,
      params: {
        moduleId,
        exportName: "syncNumbers",
        args: [3],
      },
    }));
    expect(open).toMatchObject({
      result: { streamId: "1" },
    });
    expect(openStreamCount).toBe(1);

    const values: number[] = [];
    for (let i = 0; i < 4; i++) {
      const next = await runTestEffect(dispatch({
        jsonrpc: "2.0",
        id: 10 + i,
        method: METHOD_GEN_NEXT,
        params: { streamId: "1" },
      }));
      if (i < 3) {
        expect(next).toMatchObject({ result: { kind: "yield", value: i } });
        values.push((next as { result: { value: number } }).result.value);
      } else {
        expect(next).toMatchObject({ result: { kind: "done" } });
      }
    }
    expect(values).toEqual([0, 1, 2]);

    const close = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 20,
      method: METHOD_GEN_CLOSE,
      params: { streamId: "1" },
    }));
    expect(close).toMatchObject({ result: { ok: true } });
    expect(openStreamCount).toBe(0);
  });

  test("async generator awaits next per genNext", async () => {
    clearModuleCache();
    resetGeneratorStateForTest();
    const { dispatch } = createDispatcher();
    await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 1,
      method: METHOD_INITIALIZE,
      params: initializeParams(fixtureRoot),
    }));

    const open = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 2,
      method: METHOD_GEN_OPEN,
      params: {
        moduleId,
        exportName: "asyncNumbers",
        args: [2],
      },
    }));
    const streamId = (open as { result: { streamId: string } }).result.streamId;

    const first = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 3,
      method: METHOD_GEN_NEXT,
      params: { streamId },
    }));
    expect(first).toMatchObject({ result: { kind: "yield", value: 0 } });

    const second = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 4,
      method: METHOD_GEN_NEXT,
      params: { streamId },
    }));
    expect(second).toMatchObject({ result: { kind: "yield", value: 1 } });

    const done = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 5,
      method: METHOD_GEN_NEXT,
      params: { streamId },
    }));
    expect(done).toMatchObject({ result: { kind: "done" } });

    await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 6,
      method: METHOD_GEN_CLOSE,
      params: { streamId },
    }));
    expect(openStreamCount).toBe(0);
  });

  test("empty generator returns immediate done", async () => {
    clearModuleCache();
    resetGeneratorStateForTest();
    const { dispatch } = createDispatcher();
    await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 1,
      method: METHOD_INITIALIZE,
      params: initializeParams(fixtureRoot),
    }));

    const open = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 2,
      method: METHOD_GEN_OPEN,
      params: { moduleId, exportName: "emptyGen", args: [] },
    }));
    const streamId = (open as { result: { streamId: string } }).result.streamId;

    const next = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 3,
      method: METHOD_GEN_NEXT,
      params: { streamId },
    }));
    expect(next).toMatchObject({ result: { kind: "done" } });

    await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 4,
      method: METHOD_GEN_CLOSE,
      params: { streamId },
    }));
    expect(openStreamCount).toBe(0);
  });

  test("genClose runs finally on early close", async () => {
    clearModuleCache();
    resetGeneratorStateForTest();
    const { dispatch } = createDispatcher();
    await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 1,
      method: METHOD_INITIALIZE,
      params: initializeParams(fixtureRoot),
    }));

    const open = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 2,
      method: METHOD_GEN_OPEN,
      params: { moduleId, exportName: "withFinally", args: [] },
    }));
    const streamId = (open as { result: { streamId: string } }).result.streamId;

    await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 3,
      method: METHOD_GEN_NEXT,
      params: { streamId },
    }));

    await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 4,
      method: METHOD_GEN_CLOSE,
      params: { streamId },
    }));
    expect(openStreamCount).toBe(0);
  });

  test("invalid streamId returns application error", async () => {
    clearModuleCache();
    resetGeneratorStateForTest();
    const { dispatch } = createDispatcher();
    await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 1,
      method: METHOD_INITIALIZE,
      params: initializeParams(fixtureRoot),
    }));

    const next = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 2,
      method: METHOD_GEN_NEXT,
      params: { streamId: "missing" },
    }));
    expect(next).toMatchObject({
      error: { code: Errors.APPLICATION_ERROR },
    });
  });

  test("genNextBatch returns multiple yields in one RPC", async () => {
    clearModuleCache();
    resetGeneratorStateForTest();
    const { dispatch } = createDispatcher();
    await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 1,
      method: METHOD_INITIALIZE,
      params: initializeParams(fixtureRoot),
    }));

    const open = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 2,
      method: METHOD_GEN_OPEN,
      params: {
        moduleId,
        exportName: "syncNumbers",
        args: [3],
      },
    }));
    const streamId = (open as { result: { streamId: string } }).result.streamId;

    const batch = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 3,
      method: METHOD_GEN_NEXT_BATCH,
      params: { streamId, maxItems: 5 },
    }));
    expect(batch).toMatchObject({
      result: {
        steps: [
          { kind: "yield", value: 0 },
          { kind: "yield", value: 1 },
          { kind: "yield", value: 2 },
          { kind: "done" },
        ],
      },
    });

    await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 4,
      method: METHOD_GEN_CLOSE,
      params: { streamId },
    }));
    expect(openStreamCount).toBe(0);
  });

  test("genOpen no longer returns not implemented", async () => {
    clearModuleCache();
    resetGeneratorStateForTest();
    const { dispatch } = createDispatcher();
    await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 1,
      method: METHOD_INITIALIZE,
      params: initializeParams(fixtureRoot),
    }));

    const response = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 2,
      method: METHOD_GEN_OPEN,
      params: {
        moduleId: "fixtures/missing.ts",
        exportName: "syncNumbers",
        args: [],
      },
    }));
    expect(response).not.toMatchObject({
      error: { code: Errors.METHOD_NOT_FOUND },
    });
  });
});
