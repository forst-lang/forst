import { describe, expect, test } from "bun:test";
import { PassThrough } from "node:stream";
import { createDispatcher } from "../../src/rpc/dispatcher.js";
import {
  newRequestFrame,
  ProtoFrameReader,
  writeProtoFrame,
} from "../../src/rpc/frame.js";
import { runProtoLoop } from "../../src/rpc/proto_loop.js";
import {
  METHOD_INITIALIZE,
  METHOD_PING,
  PROTOCOL_VERSION,
  WIRE_PROTOCOL_PROTO_V1,
} from "../../src/rpc/protocol.js";
import { runTestEffect } from "../helpers/run-effect.js";

describe("runProtoLoop", () => {
  test("initialize and ping over length-prefixed frames", async () => {
    const stdin = new PassThrough();
    const stdout = new PassThrough();
    const { dispatch } = createDispatcher();

    const loopDone = runTestEffect(
      runProtoLoop(stdin, stdout, { onRequest: dispatch })
    );
    const reader = new ProtoFrameReader();
    const frames: Array<{ id: number; result: unknown }> = [];

    stdout.on("data", (chunk) => {
      reader.append(Buffer.from(chunk));
      for (;;) {
        const frame = reader.tryReadFrame();
        if (frame === null) {
          break;
        }
        if (frame.response?.okJson !== undefined) {
          frames.push({
            id: frame.id,
            result: JSON.parse(new TextDecoder().decode(frame.response.okJson)),
          });
        }
      }
    });

    writeProtoFrame(
      stdin,
      newRequestFrame(1, METHOD_INITIALIZE, {
        protocolVersion: PROTOCOL_VERSION,
        boundaryRoot: "/tmp/project",
        manifest: {
          version: 1,
          boundaryRoot: "/tmp/project",
          exports: [],
        },
        supportedProtocols: [WIRE_PROTOCOL_PROTO_V1],
      })
    );

    writeProtoFrame(stdin, newRequestFrame(2, METHOD_PING, {}));
    stdin.end();

    await loopDone;

    expect(frames).toEqual([
      { id: 1, result: { ok: true, protocol: WIRE_PROTOCOL_PROTO_V1 } },
      { id: 2, result: { pong: true } },
    ]);
  });
});
