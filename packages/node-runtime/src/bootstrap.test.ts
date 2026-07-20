import { describe, expect, test, spyOn } from "bun:test";
import { Effect } from "effect";
import { ForstNodeRuntimeLayer } from "./effect/layer.js";
import { bootstrapMain } from "./bootstrap.js";
import { envSocketPath } from "./rpc/socket_server.js";
import { runTestEffect } from "../test/helpers/run-effect.js";

describe("bootstrapMain", () => {
  test("exits when FORST_NODE_SOCKET is missing on Unix", async () => {
    if (process.platform === "win32") {
      return;
    }

    const prev = process.env[envSocketPath];
    delete process.env[envSocketPath];
    const exitSpy = spyOn(process, "exit").mockImplementation((() => {}) as typeof process.exit);

    await runTestEffect(
      bootstrapMain({}).pipe(
        Effect.provide(ForstNodeRuntimeLayer),
        Effect.catchAll(() => Effect.void)
      )
    );

    expect(exitSpy).toHaveBeenCalledWith(1);
    exitSpy.mockRestore();

    if (prev === undefined) {
      delete process.env[envSocketPath];
    } else {
      process.env[envSocketPath] = prev;
    }
  });
});
