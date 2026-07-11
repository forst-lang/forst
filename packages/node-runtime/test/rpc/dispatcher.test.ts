import { describe, expect, test } from "bun:test";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { createDispatcher } from "../../src/rpc/dispatcher.js";
import {
  APPLICATION_ERROR,
  FORBIDDEN,
  METHOD_NOT_FOUND,
  NOT_IMPLEMENTED,
  NOT_INITIALIZED,
} from "../../src/rpc/errors.js";
import {
  METHOD_CALL,
  METHOD_CALL_ASYNC,
  METHOD_GEN_OPEN,
  METHOD_INITIALIZE,
  METHOD_PING,
  METHOD_SHUTDOWN,
  PROTOCOL_VERSION,
  WIRE_PROTOCOL_PROTO_V1,
} from "../../src/rpc/protocol.js";
import { clearModuleCache } from "../../src/runtime/module_cache.js";
import { runTestEffect } from "../helpers/run-effect.js";

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
    ],
  };
}

function initializeParams(boundaryRoot: string) {
  return {
    protocolVersion: PROTOCOL_VERSION,
    boundaryRoot,
    manifest: sampleManifest(boundaryRoot),
    supportedProtocols: [WIRE_PROTOCOL_PROTO_V1],
  };
}

describe("createDispatcher initialize gate", () => {
  test("rejects call before initialize", async () => {
    const { dispatch } = createDispatcher();
    const response = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 1,
      method: METHOD_CALL,
      params: {
        moduleId: syncModuleId,
        exportName: "add",
        args: [1, 2],
      },
    }));

    expect(response).toMatchObject({
      jsonrpc: "2.0",
      id: 1,
      error: { code: NOT_INITIALIZED },
    });
  });

  test("rejects ping before initialize", async () => {
    const { dispatch } = createDispatcher();
    const response = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 2,
      method: METHOD_PING,
    }));

    expect(response).toMatchObject({
      error: { code: NOT_INITIALIZED },
    });
  });
});

describe("createDispatcher method policy", () => {
  test("returns method not found for forbidden eval RPC", async () => {
    const { dispatch } = createDispatcher();
    const response = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 3,
      method: "forst.node/eval",
      params: { code: "1+1" },
    }));

    expect(response).toMatchObject({
      error: { code: METHOD_NOT_FOUND },
    });
  });

  test("handles callAsync for async exports", async () => {
    clearModuleCache();
    const asyncModuleId = "fixtures/async-payment.ts";
    const { dispatch, state } = createDispatcher();

    await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 4,
      method: METHOD_INITIALIZE,
      params: {
        protocolVersion: PROTOCOL_VERSION,
        boundaryRoot: fixtureRoot,
        manifest: {
          version: 1,
          boundaryRoot: fixtureRoot,
          exports: [
            {
              moduleId: asyncModuleId,
              name: "create",
              kind: "asyncFunction" as const,
            },
            {
              moduleId: asyncModuleId,
              name: "concurrentEcho",
              kind: "asyncFunction" as const,
            },
            {
              moduleId: asyncModuleId,
              name: "failWithError",
              kind: "asyncFunction" as const,
            },
          ],
        },
      },
    }));

    const response = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 5,
      method: METHOD_CALL_ASYNC,
      params: {
        moduleId: asyncModuleId,
        exportName: "create",
        args: [100, "USD"],
      },
    }));

    expect(response).toMatchObject({
      result: { value: { id: "pay_async", amount: 100 } },
    });
    expect(state.initialized).toBe(true);
  });

  test("callAsync preserves rejection stack metadata", async () => {
    clearModuleCache();
    const asyncModuleId = "fixtures/async-payment.ts";
    const { dispatch } = createDispatcher();
    await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 6,
      method: METHOD_INITIALIZE,
      params: {
        protocolVersion: PROTOCOL_VERSION,
        boundaryRoot: fixtureRoot,
        manifest: {
          version: 1,
          boundaryRoot: fixtureRoot,
          exports: [
            {
              moduleId: asyncModuleId,
              name: "failWithError",
              kind: "asyncFunction" as const,
            },
          ],
        },
      },
    }));

    const response = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 7,
      method: METHOD_CALL_ASYNC,
      params: {
        moduleId: asyncModuleId,
        exportName: "failWithError",
        args: [],
      },
    }));

    expect(response).toMatchObject({
      error: {
        code: APPLICATION_ERROR,
        message: "Payment provider timeout",
        data: {
          moduleId: asyncModuleId,
          exportName: "failWithError",
        },
      },
    });
    const data = (response as { error: { data: { stack?: string } } }).error.data;
    expect(typeof data.stack).toBe("string");
    expect(data.stack).toContain("failWithError");
  });

  test("genOpen rejects non-generator exports", async () => {
    const { dispatch } = createDispatcher();
    await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 6,
      method: METHOD_INITIALIZE,
      params: initializeParams(fixtureRoot),
    }));

    const response = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 7,
      method: METHOD_GEN_OPEN,
      params: {
        moduleId: syncModuleId,
        exportName: "add",
        args: [],
      },
    }));

    expect(response).toMatchObject({
      error: { code: APPLICATION_ERROR, message: "export is not a generator" },
    });
  });
});

describe("createDispatcher happy path", () => {
  test("initialize → ping → call → shutdown", async () => {
    clearModuleCache();
    const { dispatch, state } = createDispatcher();

    const init = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 10,
      method: METHOD_INITIALIZE,
      params: initializeParams(fixtureRoot),
    }));
    expect(init).toMatchObject({
      result: { ok: true, protocol: WIRE_PROTOCOL_PROTO_V1 },
    });
    expect(state.initialized).toBe(true);

    const ping = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 11,
      method: METHOD_PING,
    }));
    expect(ping).toMatchObject({ result: { pong: true } });

    const call = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 12,
      method: METHOD_CALL,
      params: {
        moduleId: syncModuleId,
        exportName: "add",
        args: [2, 3],
      },
    }));
    expect(call).toMatchObject({ result: { value: 5 } });

    const shutdown = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 13,
      method: METHOD_SHUTDOWN,
    }));
    expect(shutdown).toMatchObject({ result: { ok: true } });
    expect(state.shuttingDown).toBe(true);
  });

  test("rejects call for export not in manifest", async () => {
    clearModuleCache();
    const { dispatch } = createDispatcher();
    await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 20,
      method: METHOD_INITIALIZE,
      params: initializeParams(fixtureRoot),
    }));

    const response = await runTestEffect(dispatch({
      jsonrpc: "2.0",
      id: 21,
      method: METHOD_CALL,
      params: {
        moduleId: syncModuleId,
        exportName: "greet",
        args: ["world"],
      },
    }));

    expect(response).toMatchObject({
      error: { code: FORBIDDEN },
    });
  });
});
