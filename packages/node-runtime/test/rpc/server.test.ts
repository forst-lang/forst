import { describe, expect, test } from "bun:test";
import { PassThrough } from "node:stream";
import { Effect, Fiber } from "effect";
import { ForstNodeRuntimeLayer } from "../../src/effect/layer.js";
import { createNodeRuntimeSetup } from "../../src/effect/runtime.js";
import {
  newRequestFrame,
  ProtoFrameReader,
  writeProtoFrame,
} from "../../src/rpc/frame.js";
import {
  METHOD_INITIALIZE,
  METHOD_PING,
  METHOD_SHUTDOWN,
  PROTOCOL_VERSION,
  WIRE_PROTOCOL_PROTO_V1,
} from "../../src/rpc/protocol.js";
import { startRpcServer } from "../../src/rpc/server.js";
import { runTestEffect } from "../helpers/run-effect.js";

describe("startRpcServer", () => {
  test("handles initialize ping shutdown over stream transport", async () => {
    const input = new PassThrough();
    const output = new PassThrough();
    const setup = createNodeRuntimeSetup(ForstNodeRuntimeLayer);
    const boundaryRoot = process.cwd();

    const fiber = Effect.runFork(
      startRpcServer(input, output, {
        exitProcessOnShutdown: false,
        runtime: setup.runtime,
      }).pipe(Effect.provide(setup.layer))
    );

    const reader = new ProtoFrameReader();
    const responses: unknown[] = [];
    output.on("data", (chunk) => {
      reader.append(Buffer.from(chunk));
      for (;;) {
        const frame = reader.tryReadFrame();
        if (frame === null) {
          break;
        }
        if (frame.response?.okJson !== undefined) {
          responses.push(
            JSON.parse(new TextDecoder().decode(frame.response.okJson))
          );
        }
      }
    });

    writeProtoFrame(
      input,
      newRequestFrame(1, METHOD_INITIALIZE, {
        protocolVersion: PROTOCOL_VERSION,
        boundaryRoot,
        manifest: {
          version: 1,
          boundaryRoot,
          exports: [],
        },
        supportedProtocols: [WIRE_PROTOCOL_PROTO_V1],
      })
    );
    writeProtoFrame(input, newRequestFrame(2, METHOD_PING, {}));
    writeProtoFrame(input, newRequestFrame(3, METHOD_SHUTDOWN, {}));

    await new Promise<void>((resolve, reject) => {
      const deadline = Date.now() + 5000;
      const tick = (): void => {
        if (responses.length >= 3) {
          resolve();
          return;
        }
        if (Date.now() > deadline) {
          reject(new Error("timeout waiting for rpc responses"));
          return;
        }
        setTimeout(tick, 20);
      };
      tick();
    });

    input.end();
    await Effect.runPromise(Fiber.interrupt(fiber));
    expect(responses.length).toBeGreaterThanOrEqual(3);
  });
});
