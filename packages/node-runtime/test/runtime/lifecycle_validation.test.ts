import { describe, expect, test, afterEach } from "bun:test";
import { Effect, Either } from "effect";
import {
  PROTOCOL_VERSION,
  WIRE_PROTOCOL_PROTO_V1,
} from "../../src/rpc/protocol.js";
import {
  createRuntimeState,
  initializeRuntime,
  resetHostInitCacheForTest,
  assertInitialized,
} from "../../src/runtime/lifecycle.js";
import { runTestEffect } from "../helpers/run-effect.js";

const boundaryRoot = "/tmp/forst-boundary";

function baseParams(overrides: Record<string, unknown> = {}) {
  return {
    protocolVersion: PROTOCOL_VERSION,
    boundaryRoot,
    manifest: {
      version: 1 as const,
      boundaryRoot,
      exports: [],
    },
    supportedProtocols: [WIRE_PROTOCOL_PROTO_V1],
    ...overrides,
  };
}

describe("initializeRuntime validation", () => {
  afterEach(() => {
    resetHostInitCacheForTest();
  });

  test("rejects unsupported protocol version", async () => {
    const state = createRuntimeState();
    const result = await runTestEffect(
      initializeRuntime(state, baseParams({ protocolVersion: 999 })).pipe(
        Effect.either
      )
    );
    expect(Either.isLeft(result)).toBe(true);
  });

  test("rejects empty boundaryRoot", async () => {
    const state = createRuntimeState();
    const result = await runTestEffect(
      initializeRuntime(state, baseParams({ boundaryRoot: "" })).pipe(
        Effect.either
      )
    );
    expect(Either.isLeft(result)).toBe(true);
  });

  test("rejects manifest boundaryRoot mismatch", async () => {
    const state = createRuntimeState();
    const result = await runTestEffect(
      initializeRuntime(
        state,
        baseParams({
          manifest: {
            version: 1 as const,
            boundaryRoot: "/other/root",
            exports: [],
          },
        })
      ).pipe(Effect.either)
    );
    expect(Either.isLeft(result)).toBe(true);
  });

  test("rejects second initialize on same dispatcher state", async () => {
    const state = createRuntimeState();
    await runTestEffect(initializeRuntime(state, baseParams()));
    const result = await runTestEffect(
      initializeRuntime(state, baseParams()).pipe(Effect.either)
    );
    expect(Either.isLeft(result)).toBe(true);
  });

  test("assertInitialized throws before initialize", () => {
    const state = createRuntimeState();
    expect(() => assertInitialized(state)).toThrow();
  });
});
