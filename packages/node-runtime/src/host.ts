import * as fs from "node:fs";
import * as net from "node:net";
import * as path from "node:path";
import type { AddressInfo } from "node:net";
import { Effect, Fiber, Layer } from "effect";
import { causeToError } from "./errors/cause.js";
import { ForstNodeRuntimeLayer } from "./effect/layer.js";
import {
  createNodeRuntimeSetup,
  type ForstNodeRuntime,
} from "./effect/runtime.js";
import { startRpcServer } from "./rpc/server.js";
import * as HostErrors from "./host/errors.js";

const envHostEnabled = "FORST_NODE_HOST";
const envHostLeader = "FORST_NODE_HOST_LEADER";
const envSocketPath = "FORST_NODE_SOCKET";
const envReadyPath = "FORST_NODE_HOST_READY";

export type HostReadyPhase = "listening" | "app";

export interface HostOptions {
  socketPath?: string;
  readyPath?: string;
  /** When true, listen on the socket but defer the ready file until signalForstAppReady(). */
  deferAppReady?: boolean;
  /**
   * Effect layer for host RPC forks (logging, tracing, …).
   * Defaults to `ForstNodeRuntimeLayer`. Provide the same layer you use at the app boundary.
   */
  runtimeLayer?: Layer.Layer<never>;
}

export interface HostHandle {
  socketPath: string;
  close(): Effect.Effect<void, Error, never>;
}

interface HostCloseContext {
  socketPath: string;
  readyPath: string;
}

let startPromise: Promise<HostHandle> | null = null;
let activeServer: net.Server | null = null;
let activeConnection: net.Socket | null = null;
let activeRpcFiber: Fiber.RuntimeFiber<void, never> | null = null;
let activeReadyPath = "";
let activeSocketPath = "";
let appReadySignaled = false;
let hostRuntimeLayer: Layer.Layer<never> = ForstNodeRuntimeLayer;
let hostRuntime: ForstNodeRuntime = createNodeRuntimeSetup(
  ForstNodeRuntimeLayer
).runtime;

function hostEnabled(): boolean {
  return process.env[envHostEnabled] === "1";
}

let hostLeaderOverrideForTest: boolean | null = null;

function registerPreloaded(): boolean {
  const argv = process.execArgv;
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i] ?? "";
    if (arg === "--import" && argv[i + 1]?.includes("register.mjs")) {
      return true;
    }
    if (arg.includes("register.mjs")) {
      return true;
    }
  }
  return false;
}

function hostLeaderEnabled(): boolean {
  if (hostLeaderOverrideForTest != null) {
    return hostLeaderOverrideForTest;
  }
  if (process.env[envHostLeader] !== "1") {
    return false;
  }
  return registerPreloaded();
}

interface HostReadyMarker {
  pid?: number;
  socket?: string;
  phase?: string;
}

function readReadyMarker(readyPath: string): HostReadyMarker | null {
  try {
    return JSON.parse(fs.readFileSync(readyPath, "utf8")) as HostReadyMarker;
  } catch {
    return null;
  }
}

function processAlive(pid: number): boolean {
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

function isWindows(): boolean {
  return process.platform === "win32";
}

function writeReadyFile(
  readyPath: string,
  socketPath: string,
  phase: HostReadyPhase
): void {
  writeReadyFilePayload(readyPath, {
    pid: process.pid,
    socket: socketPath,
    phase,
    tcpPort: isWindows() ? Number(socketPath.split(":").pop()) : undefined,
  });
}

function writeReadyFilePayload(
  readyPath: string,
  payload: Record<string, unknown>
): void {
  fs.writeFileSync(readyPath, `${JSON.stringify(payload)}\n`, { encoding: "utf8" });
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

function listenUnix(socketPath: string): Effect.Effect<net.Server, Error, never> {
  return Effect.async<net.Server, Error>((resume) => {
    const dir = path.dirname(socketPath);
    const readyPath = process.env[envReadyPath] ?? "";
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
                `host already running (pid=${marker.pid}, socket=${socketPath})`
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
              `host socket already in use (socket=${socketPath}); another host process may be running`
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

function listenTcp(): Effect.Effect<
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

function attachConnectionHandler(server: net.Server): void {
  server.on("connection", (conn) => {
    if (activeConnection) {
      void Effect.runFork(
        Effect.logWarning("host_reject_duplicate_client").pipe(
          Effect.annotateLogs({
            event: "host_reject_duplicate_client",
            pid: process.pid,
          }),
          Effect.provide(hostRuntimeLayer)
        )
      );
      conn.destroy();
      return;
    }
    activeConnection = conn;
    void Effect.runFork(
      Effect.logInfo("host_client_connected").pipe(
        Effect.annotateLogs({ event: "host_client_connected", pid: process.pid }),
        Effect.provide(hostRuntimeLayer)
      )
    );
    conn.on("close", () => {
      if (activeConnection === conn) {
        activeConnection = null;
        if (activeRpcFiber) {
          void Effect.runPromise(Fiber.interrupt(activeRpcFiber));
          activeRpcFiber = null;
        }
      }
    });
    activeRpcFiber = Effect.runFork(
      startRpcServer(conn, conn, {
        exitProcessOnShutdown: false,
        runtime: hostRuntime,
      }).pipe(
        Effect.provide(hostRuntimeLayer),
        Effect.catchAllDefect((cause) =>
          Effect.logError("host_rpc_fatal").pipe(
            Effect.annotateLogs({
              event: "host_rpc_fatal",
              message: cause instanceof Error ? cause.message : String(cause),
            })
          )
        )
      )
    );
  });
}

export function resetHostForTest(): void {
  if (activeRpcFiber) {
    void Effect.runPromise(Fiber.interrupt(activeRpcFiber));
    activeRpcFiber = null;
  }
  if (activeConnection) {
    activeConnection.destroy();
    activeConnection = null;
  }
  if (activeServer) {
    activeServer.close();
    activeServer = null;
  }
  startPromise = null;
  activeReadyPath = "";
  activeSocketPath = "";
  appReadySignaled = false;
  hostRuntimeLayer = ForstNodeRuntimeLayer;
  hostRuntime = createNodeRuntimeSetup(ForstNodeRuntimeLayer).runtime;
  hostLeaderOverrideForTest = null;
}

/** Test-only: bypass register preload check for direct startForstNodeHost calls. */
export function setHostLeaderOverrideForTest(value: boolean | null): void {
  hostLeaderOverrideForTest = value;
}

const closeHostHandle = Effect.fn("Host.close")(function* (ctx: HostCloseContext) {
  yield* Effect.annotateCurrentSpan("socket", ctx.socketPath);
  if (activeRpcFiber) {
    yield* Fiber.interrupt(activeRpcFiber);
    activeRpcFiber = null;
  }
  yield* Effect.tryPromise({
    try: async () => {
      if (activeConnection) {
        activeConnection.destroy();
        activeConnection = null;
      }
      await new Promise<void>((resolve, reject) => {
        if (!activeServer) {
          resolve();
          return;
        }
        activeServer.close((err) => {
          if (err) {
            reject(err);
            return;
          }
          resolve();
        });
      });
      activeServer = null;
      if (!isWindows() && ctx.socketPath && fs.existsSync(ctx.socketPath)) {
        try {
          fs.unlinkSync(ctx.socketPath);
        } catch {
          // best effort
        }
      }
      if (ctx.readyPath && fs.existsSync(ctx.readyPath)) {
        try {
          fs.unlinkSync(ctx.readyPath);
        } catch {
          // best effort
        }
      }
      activeReadyPath = "";
      activeSocketPath = "";
      appReadySignaled = false;
    },
    catch: (cause) => causeToError(cause),
  });
});

const hostStartNonLeaderNoop = Effect.fn("Host.startNonLeaderNoop")(function* () {
  yield* Effect.logDebug("host_skip_non_leader").pipe(
    Effect.annotateLogs({
      event: "host_skip_non_leader",
      reason: "FORST_NODE_HOST_LEADER unset",
      pid: process.pid,
    })
  );
  return {
    socketPath: "",
    close: () => Effect.void,
  } satisfies HostHandle;
});

const hostStartNoop = Effect.fn("Host.startNoop")(function* () {
  yield* Effect.logDebug("host_skip").pipe(
    Effect.annotateLogs({ event: "host_skip", reason: "FORST_NODE_HOST unset" })
  );
  return {
    socketPath: "",
    close: () => Effect.void,
  } satisfies HostHandle;
});

const hostStart = Effect.fn("Host.start")(function* (options: HostOptions) {
  const readyPath = options.readyPath ?? process.env[envReadyPath] ?? "";
  let socketPath = options.socketPath ?? process.env[envSocketPath] ?? "";
  const deferAppReady = options.deferAppReady ?? false;

  let server: net.Server;
  if (isWindows()) {
    const tcp = yield* listenTcp();
    server = tcp.server;
    socketPath = `tcp://127.0.0.1:${tcp.port}`;
  } else {
    if (!socketPath) {
      return yield* Effect.fail(HostErrors.hostSocketRequired());
    }
    server = yield* listenUnix(socketPath);
  }

  activeServer = server;
  activeReadyPath = readyPath;
  activeSocketPath = socketPath;
  attachConnectionHandler(server);

  if (readyPath && !deferAppReady) {
    yield* Effect.sync(() => {
      writeReadyFile(readyPath, socketPath, "app");
      appReadySignaled = true;
    });
  }

  yield* Effect.annotateCurrentSpan("socket", socketPath);
  yield* Effect.annotateCurrentSpan("defer_app_ready", deferAppReady);

  yield* Effect.logInfo("host_listening").pipe(
    Effect.annotateLogs({
      event: "host_listening",
      socket: socketPath,
      pid: process.pid,
      defer_app_ready: deferAppReady,
    })
  );

  const closeCtx: HostCloseContext = { socketPath, readyPath };
  return {
    socketPath,
    close: () => closeHostHandle(closeCtx),
  } satisfies HostHandle;
});

/** Marks app initialization complete and writes the ready file with phase "app". */
export const signalForstAppReady = Effect.fn("Host.signalAppReady")(function* () {
  if (!hostEnabled()) {
    yield* Effect.logDebug("host_app_ready_skip").pipe(
      Effect.annotateLogs({
        event: "host_app_ready_skip",
        reason: "FORST_NODE_HOST unset",
      })
    );
    return;
  }
  if (appReadySignaled) {
    return;
  }
  if (!activeReadyPath) {
    return yield* Effect.fail(HostErrors.hostReadyPathUnset());
  }
  if (!activeSocketPath) {
    return yield* Effect.fail(HostErrors.hostSocketPathUnset());
  }

  yield* Effect.sync(() => {
    writeReadyFile(activeReadyPath, activeSocketPath, "app");
    appReadySignaled = true;
  });
  yield* Effect.annotateCurrentSpan("socket", activeSocketPath);
  yield* Effect.logInfo("host_app_ready").pipe(
    Effect.annotateLogs({
      event: "host_app_ready",
      socket: activeSocketPath,
      pid: process.pid,
    })
  );
});

/**
 * No-op unless FORST_NODE_HOST=1. Idempotent. Logs to stderr only. Single client connection.
 */
export function startForstNodeHost(
  options: HostOptions = {}
): Effect.Effect<HostHandle, Error, never> {
  if (!hostEnabled()) {
    return hostStartNoop();
  }

  if (!hostLeaderEnabled()) {
    return hostStartNonLeaderNoop();
  }

  if (startPromise) {
    return Effect.tryPromise({
      try: () => startPromise!,
      catch: (cause) => causeToError(cause),
    });
  }

  const setup = createNodeRuntimeSetup(
    options.runtimeLayer ?? ForstNodeRuntimeLayer
  );
  hostRuntimeLayer = setup.layer;
  hostRuntime = setup.runtime;

  startPromise = Effect.runPromise(
    hostStart(options).pipe(Effect.provide(hostRuntimeLayer))
  );

  return Effect.tryPromise({
    try: () => startPromise!,
    catch: (cause) => causeToError(cause),
  });
}
