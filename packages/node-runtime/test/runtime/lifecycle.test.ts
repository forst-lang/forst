import { describe, expect, test } from "bun:test";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { createDispatcher } from "../../src/rpc/dispatcher.js";
import {
  METHOD_INITIALIZE,
  PROTOCOL_VERSION,
  WIRE_PROTOCOL_PROTO_V1,
} from "../../src/rpc/protocol.js";
import { resetHostInitCacheForTest } from "../../src/runtime/lifecycle.js";
import { runTestEffect } from "../helpers/run-effect.js";

const testDir = path.dirname(fileURLToPath(import.meta.url));
const fixtureRoot = path.resolve(testDir, "..");

function initializeParams(boundaryRoot: string) {
  return {
    protocolVersion: PROTOCOL_VERSION,
    boundaryRoot,
    manifest: {
      version: 1 as const,
      boundaryRoot,
      exports: [
        {
          moduleId: "fixtures/sync-add.ts",
          name: "add",
          kind: "function" as const,
        },
      ],
    },
    supportedProtocols: [WIRE_PROTOCOL_PROTO_V1],
  };
}

describe("initializeRuntime host cache", () => {
  test("reuses manifest snapshot across reconnect dispatchers", async () => {
    resetHostInitCacheForTest();
    const params = initializeParams(fixtureRoot);

    const first = createDispatcher();
    const init1 = await runTestEffect(first.dispatch({
      jsonrpc: "2.0",
      id: 1,
      method: METHOD_INITIALIZE,
      params,
    }));
    expect(init1).toMatchObject({ result: { ok: true } });
    expect(first.state.initialized).toBe(true);

    const second = createDispatcher();
    const init2 = await runTestEffect(second.dispatch({
      jsonrpc: "2.0",
      id: 2,
      method: METHOD_INITIALIZE,
      params,
    }));
    expect(init2).toMatchObject({ result: { ok: true } });
    expect(second.state.initialized).toBe(true);
    expect(second.state.index).toBe(first.state.index);
  });
});
