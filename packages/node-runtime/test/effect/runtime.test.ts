import { describe, expect, test } from "bun:test";
import { Effect, Layer, Logger, LogLevel } from "effect";
import { makeForstNodeRuntimeLayer } from "../../src/effect/layer.js";
import {
  createNodeRuntimeSetup,
  defaultNodeRuntimeSetup,
  makeForstNodeRuntime,
} from "../../src/effect/runtime.js";

describe("createNodeRuntimeSetup", () => {
  test("default setup bundles layer and runtime", () => {
    expect(defaultNodeRuntimeSetup.layer).toBeDefined();
    expect(defaultNodeRuntimeSetup.runtime).toBeDefined();
  });

  test("custom layer produces a distinct runtime", () => {
    const customLayer = Layer.mergeAll(
      makeForstNodeRuntimeLayer(),
      Logger.minimumLogLevel(LogLevel.Debug)
    );
    const setup = createNodeRuntimeSetup(customLayer);
    expect(setup.layer).toBe(customLayer);
    expect(setup.runtime).not.toBe(defaultNodeRuntimeSetup.runtime);
    expect(makeForstNodeRuntime(customLayer)).not.toBe(
      defaultNodeRuntimeSetup.runtime
    );
  });

  test("runtime runs effects with the provided layer", async () => {
    const setup = createNodeRuntimeSetup(makeForstNodeRuntimeLayer());
    await expect(
      setup.runtime.runPromise(Effect.succeed(1))
    ).resolves.toBe(1);
  });
});
