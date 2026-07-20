import { describe, expect, test, afterEach } from "bun:test";
import * as fs from "node:fs";
import * as net from "node:net";
import * as os from "node:os";
import * as path from "node:path";
import { Effect } from "effect";
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
  PROTOCOL_VERSION,
  WIRE_PROTOCOL_PROTO_V1,
} from "../../src/rpc/protocol.js";
import {
  readReadyMarker,
  startSocketRpcServer,
} from "../../src/rpc/socket_server.js";
import { runTestEffect } from "../helpers/run-effect.js";

function tempDir(): string {
  return fs.mkdtempSync(path.join(os.tmpdir(), "forst-socket-server-test-"));
}

describe("startSocketRpcServer", () => {
  afterEach(() => {
    // handles closed per test
  });

  test("writes ready marker and accepts ping", async () => {
    if (process.platform === "win32") {
      return;
    }
    const dir = tempDir();
    const socketPath = path.join(dir, "node.sock");
    const readyPath = path.join(dir, "node.sock.ready");
    const setup = createNodeRuntimeSetup(ForstNodeRuntimeLayer);

    const handle = await runTestEffect(
      startSocketRpcServer({
        socketPath,
        readyPath,
        deferAppReady: false,
        exitProcessOnShutdown: false,
        runtime: setup.runtime,
        runtimeLayer: setup.layer,
        logPrefix: "test",
      }).pipe(Effect.provide(setup.layer))
    );

    expect(fs.existsSync(readyPath)).toBe(true);
    const marker = readReadyMarker(readyPath);
    expect(marker?.phase).toBe("app");
    expect(marker?.socket).toBe(socketPath);

    const conn = net.createConnection(socketPath);
    await new Promise<void>((resolve, reject) => {
      conn.once("connect", () => resolve());
      conn.once("error", reject);
    });

    const reader = new ProtoFrameReader();
    const responses: unknown[] = [];
    conn.on("data", (chunk) => {
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
      conn,
      newRequestFrame(1, METHOD_INITIALIZE, {
        protocolVersion: PROTOCOL_VERSION,
        boundaryRoot: dir,
        manifest: {
          version: 1,
          boundaryRoot: dir,
          exports: [],
        },
        supportedProtocols: [WIRE_PROTOCOL_PROTO_V1],
      })
    );
    writeProtoFrame(conn, newRequestFrame(2, METHOD_PING, {}));

    await new Promise<void>((resolve, reject) => {
      const deadline = Date.now() + 5000;
      const tick = (): void => {
        if (responses.length >= 2) {
          resolve();
          return;
        }
        if (Date.now() > deadline) {
          reject(new Error("timeout waiting for ping response"));
          return;
        }
        setTimeout(tick, 20);
      };
      tick();
    });

    expect(responses.length).toBeGreaterThanOrEqual(2);
    conn.destroy();
    await runTestEffect(handle.close());
  });

  test("rejects duplicate client connection", async () => {
    if (process.platform === "win32") {
      return;
    }
    const dir = tempDir();
    const socketPath = path.join(dir, "node.sock");
    const setup = createNodeRuntimeSetup(ForstNodeRuntimeLayer);

    const handle = await runTestEffect(
      startSocketRpcServer({
        socketPath,
        deferAppReady: false,
        exitProcessOnShutdown: false,
        runtime: setup.runtime,
        runtimeLayer: setup.layer,
        logPrefix: "test",
      }).pipe(Effect.provide(setup.layer))
    );

    const first = net.createConnection(socketPath);
    await new Promise<void>((resolve, reject) => {
      first.once("connect", () => resolve());
      first.once("error", reject);
    });

    const second = net.createConnection(socketPath);
    const destroyed = await new Promise<boolean>((resolve) => {
      second.once("close", () => resolve(second.destroyed));
      second.once("error", () => resolve(true));
      setTimeout(() => resolve(second.destroyed), 200);
    });
    expect(destroyed).toBe(true);

    first.destroy();
    await runTestEffect(handle.close());
    expect(fs.existsSync(socketPath)).toBe(false);
  });
});
