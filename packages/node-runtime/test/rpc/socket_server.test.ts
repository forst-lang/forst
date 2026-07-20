import { describe, expect, test, afterEach } from "bun:test";
import { spawn } from "node:child_process";
import * as fs from "node:fs";
import * as net from "node:net";
import * as os from "node:os";
import * as path from "node:path";
import { Effect, Either } from "effect";
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
  envSocketPath,
  listenTcp,
  listenUnix,
  readReadyMarker,
  resetSocketRpcServersForTest,
  startSocketRpcServer,
} from "../../src/rpc/socket_server.js";
import { runTestEffect } from "../helpers/run-effect.js";

function tempDir(): string {
  return fs.mkdtempSync(path.join(os.tmpdir(), "forst-socket-server-test-"));
}

function serverSetup() {
  return createNodeRuntimeSetup(ForstNodeRuntimeLayer);
}

function startTestServer(
  setup: ReturnType<typeof serverSetup>,
  options: {
    socketPath: string;
    readyPath?: string;
    deferAppReady?: boolean;
  }
) {
  return runTestEffect(
    startSocketRpcServer({
      socketPath: options.socketPath,
      readyPath: options.readyPath,
      deferAppReady: options.deferAppReady ?? false,
      exitProcessOnShutdown: false,
      runtime: setup.runtime,
      runtimeLayer: setup.layer,
      logPrefix: "test",
    }).pipe(Effect.provide(setup.layer))
  );
}

describe("readReadyMarker", () => {
  test("returns null for missing file", () => {
    expect(readReadyMarker("/nonexistent/ready.json")).toBeNull();
  });

  test("returns null for corrupt JSON", () => {
    const dir = tempDir();
    const readyPath = path.join(dir, "bad.ready");
    fs.writeFileSync(readyPath, "not-json");
    expect(readReadyMarker(readyPath)).toBeNull();
  });
});

describe("listenTcp", () => {
  test("binds on loopback with a positive port", async () => {
    const tcp = await runTestEffect(listenTcp());
    expect(tcp.port).toBeGreaterThan(0);
    await new Promise<void>((resolve, reject) => {
      tcp.server.close((err) => (err ? reject(err) : resolve()));
    });
  });
});

describe("startSocketRpcServer", () => {
  afterEach(() => {
    resetSocketRpcServersForTest();
  });

  test("writes ready marker and accepts ping", async () => {
    if (process.platform === "win32") {
      return;
    }
    const dir = tempDir();
    const socketPath = path.join(dir, "node.sock");
    const readyPath = path.join(dir, "node.sock.ready");
    const setup = serverSetup();

    const handle = await startTestServer(setup, { socketPath, readyPath });

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
    const setup = serverSetup();

    const handle = await startTestServer(setup, { socketPath });

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

  test("rejects listen when ready marker names a live pid", async () => {
    if (process.platform === "win32") {
      return;
    }
    const dir = tempDir();
    const socketPath = path.join(dir, "node.sock");
    const readyPath = path.join(dir, "node.sock.ready");
    const sleeper = spawn(process.execPath, ["-e", "setInterval(() => {}, 1000)"], {
      stdio: "ignore",
    });
    fs.writeFileSync(
      readyPath,
      `${JSON.stringify({ pid: sleeper.pid, socket: socketPath, phase: "app" })}\n`
    );

    const result = await runTestEffect(
      listenUnix(socketPath, readyPath).pipe(Effect.either)
    );
    sleeper.kill("SIGTERM");

    expect(Either.isLeft(result)).toBe(true);
    if (Either.isLeft(result)) {
      expect(result.left.message).toContain("already running");
    }
  });

  test("ignores stale ready marker with dead pid and stale socket file", async () => {
    if (process.platform === "win32") {
      return;
    }
    const dir = tempDir();
    const socketPath = path.join(dir, "node.sock");
    const readyPath = path.join(dir, "node.sock.ready");
    fs.writeFileSync(
      readyPath,
      `${JSON.stringify({ pid: 999999999, socket: socketPath, phase: "app" })}\n`
    );
    fs.writeFileSync(socketPath, "stale");

    const setup = serverSetup();
    const handle = await startTestServer(setup, { socketPath, readyPath });
    expect(fs.existsSync(readyPath)).toBe(true);
    expect(readReadyMarker(readyPath)?.pid).toBe(process.pid);
    await runTestEffect(handle.close());
  });

  test("fails listen when socket parent path is not a directory", async () => {
    if (process.platform === "win32") {
      return;
    }
    const dir = tempDir();
    const blockerFile = path.join(dir, "not-a-dir");
    fs.writeFileSync(blockerFile, "blocked");
    const socketPath = path.join(blockerFile, "node.sock");

    const result = await runTestEffect(
      listenUnix(socketPath, "").pipe(Effect.either)
    );

    expect(Either.isLeft(result)).toBe(true);
  });

  test("second bind on held socket path fails or is rejected", async () => {
    if (process.platform === "win32") {
      return;
    }
    const dir = tempDir();
    const socketPath = path.join(dir, "node.sock");
    const setup = serverSetup();

    const first = await startTestServer(setup, { socketPath });
    const second = await runTestEffect(
      listenUnix(socketPath, "").pipe(Effect.either)
    );
    await runTestEffect(first.close());

    if (Either.isLeft(second)) {
      expect(second.left.message).toMatch(/already in use|Failed to listen/i);
      return;
    }
    await new Promise<void>((resolve, reject) => {
      second.right.close((err) => (err ? reject(err) : resolve()));
    });
  });

  test("requires FORST_NODE_SOCKET on Unix when path unset", async () => {
    if (process.platform === "win32") {
      return;
    }
    const prev = process.env[envSocketPath];
    delete process.env[envSocketPath];
    const setup = serverSetup();

    const result = await runTestEffect(
      startSocketRpcServer({
        deferAppReady: false,
        exitProcessOnShutdown: false,
        runtime: setup.runtime,
        runtimeLayer: setup.layer,
        logPrefix: "test",
      })
        .pipe(Effect.either, Effect.provide(setup.layer))
    );

    if (prev === undefined) {
      delete process.env[envSocketPath];
    } else {
      process.env[envSocketPath] = prev;
    }

    expect(Either.isLeft(result)).toBe(true);
    if (Either.isLeft(result)) {
      expect(result.left.message).toContain("FORST_NODE_SOCKET is required on Unix");
    }
  });

  test("deferAppReady writes ready file only after signalReady", async () => {
    if (process.platform === "win32") {
      return;
    }
    const dir = tempDir();
    const socketPath = path.join(dir, "node.sock");
    const readyPath = path.join(dir, "node.sock.ready");
    const setup = serverSetup();

    const handle = await startTestServer(setup, {
      socketPath,
      readyPath,
      deferAppReady: true,
    });
    expect(fs.existsSync(readyPath)).toBe(false);

    handle.signalReady();
    expect(fs.existsSync(readyPath)).toBe(true);
    const first = readReadyMarker(readyPath);
    expect(first?.phase).toBe("app");

    handle.signalReady();
    const second = readReadyMarker(readyPath);
    expect(second).toEqual(first);

    await runTestEffect(handle.close());
  });

  test("close removes ready file and is safe to call twice", async () => {
    if (process.platform === "win32") {
      return;
    }
    const dir = tempDir();
    const socketPath = path.join(dir, "node.sock");
    const readyPath = path.join(dir, "node.sock.ready");
    const setup = serverSetup();

    const handle = await startTestServer(setup, { socketPath, readyPath });
    expect(fs.existsSync(readyPath)).toBe(true);

    await runTestEffect(handle.close());
    expect(fs.existsSync(socketPath)).toBe(false);
    expect(fs.existsSync(readyPath)).toBe(false);

    await runTestEffect(handle.close().pipe(Effect.either));
  });

  test("resetSocketRpcServersForTest closes multiple servers", async () => {
    if (process.platform === "win32") {
      return;
    }
    const dir = tempDir();
    const socketPathA = path.join(dir, "a.sock");
    const readyPathA = path.join(dir, "a.sock.ready");
    const socketPathB = path.join(dir, "b.sock");
    const readyPathB = path.join(dir, "b.sock.ready");
    const setup = serverSetup();

    await startTestServer(setup, { socketPath: socketPathA, readyPath: readyPathA });
    await startTestServer(setup, { socketPath: socketPathB, readyPath: readyPathB });

    resetSocketRpcServersForTest();

    expect(fs.existsSync(socketPathA)).toBe(false);
    expect(fs.existsSync(readyPathA)).toBe(false);
    expect(fs.existsSync(socketPathB)).toBe(false);
    expect(fs.existsSync(readyPathB)).toBe(false);
  });
});
