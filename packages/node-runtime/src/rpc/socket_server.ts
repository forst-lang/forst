import * as fs from "node:fs";
import * as net from "node:net";
import * as path from "node:path";
import type { AddressInfo } from "node:net";
import { Effect, Fiber, Layer } from "effect";
import { causeToError } from "../errors/cause.js";
import type { ForstNodeRuntime } from "../effect/runtime.js";
import { startRpcServer } from "./server.js";

export const envSocketPath = "FORST_NODE_SOCKET";
export const envReadyPath = "FORST_NODE_HOST_READY";

/** Phase recorded in the host/bootstrap ready file. */
export type ReadyPhase = "listening" | "app";

export interface ReadyMarker {
  pid?: number;
  socket?: string;
  phase?: string;
  tcpPort?: number;
}

export interface SocketRpcOptions {
  socketPath?: string;
  readyPath?: string;
  deferAppReady?: boolean;
  exitProcessOnShutdown: boolean;
  runtime: ForstNodeRuntime;
  runtimeLayer: Layer.Layer<never>;
  /** Log event name prefix (e.g. "host" or "bootstrap"). */
  logPrefix?: string;
}

export interface SocketRpcHandle {
  socketPath: string;
  readyPath: string;
  signalReady(): void;
  close(): Effect.Effect<void, Error, never>;
}

interface SocketServerState {
  server: net.Server;
  activeConnection: net.Socket | null;
  activeRpcFiber: Fiber.RuntimeFiber<void, never> | null;
  appReadySignaled: boolean;
  socketPath: string;
  readyPath: string;
}

const activeStates = new Set<SocketServerState>();

/** Test-only: synchronously closes all socket RPC servers. */
export function resetSocketRpcServersForTest(): void {
  for (const state of activeStates) {
    if (state.activeRpcFiber) {
      void Effect.runPromise(Fiber.interrupt(state.activeRpcFiber));
      state.activeRpcFiber = null;
    }
    if (state.activeConnection) {
      state.activeConnection.destroy();
      state.activeConnection = null;
    }
    try {
      state.server.close();
    } catch {
      // best effort
    }
    if (!isWindows() && state.socketPath && fs.existsSync(state.socketPath)) {
      try {
        fs.unlinkSync(state.socketPath);
      } catch {
        // best effort
      }
    }
    if (state.readyPath && fs.existsSync(state.readyPath)) {
      try {
        fs.unlinkSync(state.readyPath);
      } catch {
        // best effort
      }
    }
    state.appReadySignaled = false;
  }
  activeStates.clear();
}

export function isWindows(): boolean {
  return process.platform === "win32";
}

export function processAlive(pid: number): boolean {
  if (pid <= 0) {
    return false;
  }
  try {
    process.kill(pid, 0);
    return true;
  } catch {
    return false;
  }
}

export function readReadyMarker(readyPath: string): ReadyMarker | null {
  try {
    return JSON.parse(fs.readFileSync(readyPath, "utf8")) as ReadyMarker;
  } catch {
    return null;
  }
}

export function writeReadyFile(
  readyPath: string,
  socketPath: string,
  phase: ReadyPhase
): void {
  writeReadyFilePayload(readyPath, {
    pid: process.pid,
    socket: socketPath,
    phase,
    tcpPort: isWindows() ? Number(socketPath.split(":").pop()) : undefined,
  });
}

export function writeReadyFilePayload(
  readyPath: string,
  payload: Record<string, unknown>
): void {
  fs.writeFileSync(readyPath, `${JSON.stringify(payload)}\n`, {
    encoding: "utf8",
  });
}

function chmodSocket(socketPath: string): void {
  if (isWindows()) {
    return;
  }
  try {
    fs.chmodSync(socketPath, 0o600);
  } catch {
    // best effort
  }
}

export function listenUnix(
  socketPath: string,
  readyPath: string
): Effect.Effect<net.Server, Error, never> {
  return Effect.async<net.Server, Error>((resume) => {
    const dir = path.dirname(socketPath);
    try {
      fs.mkdirSync(dir, { recursive: true, mode: 0o750 });
      if (readyPath) {
        const marker = readReadyMarker(readyPath);
        if (
          marker?.pid &&
          marker.pid !== process.pid &&
          processAlive(marker.pid)
        ) {
          resume(
            Effect.fail(
              new Error(
                `socket server already running (pid=${marker.pid}, socket=${socketPath})`
              )
            )
          );
          return;
        }
      }
      if (fs.existsSync(socketPath)) {
        try {
          fs.unlinkSync(socketPath);
        } catch {
          // stale socket may be removed by Go before spawn
        }
      }
    } catch (err) {
      resume(Effect.fail(causeToError(err)));
      return;
    }

    const server = net.createServer();
    server.on("error", (err) => {
      const code = (err as NodeJS.ErrnoException).code;
      if (code === "EADDRINUSE") {
        resume(
          Effect.fail(
            new Error(
              `socket already in use (socket=${socketPath}); another process may be running`
            )
          )
        );
        return;
      }
      resume(Effect.fail(err));
    });
    server.listen(socketPath, () => {
      chmodSocket(socketPath);
      resume(Effect.succeed(server));
    });
  });
}

export function listenTcp(): Effect.Effect<
  { server: net.Server; port: number },
  Error,
  never
> {
  return Effect.async((resume) => {
    const server = net.createServer();
    server.on("error", (err) => resume(Effect.fail(err)));
    server.listen(0, "127.0.0.1", () => {
      const addr = server.address() as AddressInfo;
      resume(Effect.succeed({ server, port: addr.port }));
    });
  });
}

function attachConnectionHandler(
  state: SocketServerState,
  options: SocketRpcOptions
): void {
  const prefix = options.logPrefix ?? "socket";
  state.server.on("connection", (conn) => {
    if (state.activeConnection) {
      void Effect.runFork(
        Effect.logWarning(`${prefix}_reject_duplicate_client`).pipe(
          Effect.annotateLogs({
            event: `${prefix}_reject_duplicate_client`,
            pid: process.pid,
          }),
          Effect.provide(options.runtimeLayer)
        )
      );
      conn.destroy();
      return;
    }
    state.activeConnection = conn;
    void Effect.runFork(
      Effect.logInfo(`${prefix}_client_connected`).pipe(
        Effect.annotateLogs({
          event: `${prefix}_client_connected`,
          pid: process.pid,
        }),
        Effect.provide(options.runtimeLayer)
      )
    );
    conn.on("close", () => {
      if (state.activeConnection === conn) {
        state.activeConnection = null;
        if (state.activeRpcFiber) {
          void Effect.runPromise(Fiber.interrupt(state.activeRpcFiber));
          state.activeRpcFiber = null;
        }
      }
    });
    state.activeRpcFiber = Effect.runFork(
      startRpcServer(conn, conn, {
        exitProcessOnShutdown: options.exitProcessOnShutdown,
        runtime: options.runtime,
      }).pipe(
        Effect.provide(options.runtimeLayer),
        Effect.catchAllDefect((cause) =>
          Effect.logError(`${prefix}_rpc_fatal`).pipe(
            Effect.annotateLogs({
              event: `${prefix}_rpc_fatal`,
              message: cause instanceof Error ? cause.message : String(cause),
            })
          )
        )
      )
    );
  });
}

const closeSocketServer = Effect.fn("SocketServer.close")(function* (
  state: SocketServerState
) {
  yield* Effect.annotateCurrentSpan("socket", state.socketPath);
  activeStates.delete(state);
  if (state.activeRpcFiber) {
    yield* Fiber.interrupt(state.activeRpcFiber);
    state.activeRpcFiber = null;
  }
  yield* Effect.tryPromise({
    try: async () => {
      if (state.activeConnection) {
        state.activeConnection.destroy();
        state.activeConnection = null;
      }
      await new Promise<void>((resolve, reject) => {
        state.server.close((err) => {
          if (err) {
            reject(err);
            return;
          }
          resolve();
        });
      });
      if (!isWindows() && state.socketPath && fs.existsSync(state.socketPath)) {
        try {
          fs.unlinkSync(state.socketPath);
        } catch {
          // best effort
        }
      }
      if (state.readyPath && fs.existsSync(state.readyPath)) {
        try {
          fs.unlinkSync(state.readyPath);
        } catch {
          // best effort
        }
      }
    },
    catch: (cause) => causeToError(cause),
  });
});

export const startSocketRpcServer = Effect.fn("SocketServer.start")(
  function* (options: SocketRpcOptions) {
    const readyPath =
      options.readyPath ?? process.env[envReadyPath] ?? "";
    let socketPath =
      options.socketPath ?? process.env[envSocketPath] ?? "";
    const deferAppReady = options.deferAppReady ?? false;
    const prefix = options.logPrefix ?? "socket";

    let server: net.Server;
    if (isWindows()) {
      const tcp = yield* listenTcp();
      server = tcp.server;
      socketPath = `tcp://127.0.0.1:${tcp.port}`;
    } else {
      if (!socketPath) {
        return yield* Effect.fail(
          new Error("FORST_NODE_SOCKET is required on Unix")
        );
      }
      server = yield* listenUnix(socketPath, readyPath);
    }

    const state: SocketServerState = {
      server,
      activeConnection: null,
      activeRpcFiber: null,
      appReadySignaled: false,
      socketPath,
      readyPath,
    };

    attachConnectionHandler(state, options);

    activeStates.add(state);

    const signalReady = (): void => {
      if (state.appReadySignaled || !state.readyPath) {
        return;
      }
      writeReadyFile(state.readyPath, state.socketPath, "app");
      state.appReadySignaled = true;
    };

    if (readyPath && !deferAppReady) {
      yield* Effect.sync(() => signalReady());
    }

    yield* Effect.annotateCurrentSpan("socket", socketPath);
    yield* Effect.annotateCurrentSpan("defer_app_ready", deferAppReady);

    yield* Effect.logInfo(`${prefix}_listening`).pipe(
      Effect.annotateLogs({
        event: `${prefix}_listening`,
        socket: socketPath,
        pid: process.pid,
        defer_app_ready: deferAppReady,
      })
    );

    return {
      socketPath,
      readyPath,
      signalReady,
      close: () => closeSocketServer(state),
    } satisfies SocketRpcHandle;
  }
);
