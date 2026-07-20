import { Effect, Layer } from "effect";
import { causeToError } from "./errors/cause.js";
import { ForstNodeRuntimeLayer } from "./effect/layer.js";
import { createNodeRuntimeSetup } from "./effect/runtime.js";
import { resetHostInitCacheForTest } from "./runtime/lifecycle.js";
import * as HostErrors from "./host/errors.js";
import {
  envReadyPath,
  envSocketPath,
  resetSocketRpcServersForTest,
  startSocketRpcServer,
  type ReadyPhase,
  type SocketRpcHandle,
} from "./rpc/socket_server.js";

const envHostEnabled = "FORST_NODE_HOST";
const envHostLeader = "FORST_NODE_HOST_LEADER";

export type HostReadyPhase = ReadyPhase;

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

let startPromise: Promise<HostHandle> | null = null;
let activeHandle: SocketRpcHandle | null = null;
let appReadySignaled = false;
let hostRuntimeLayer: Layer.Layer<never> = ForstNodeRuntimeLayer;

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

export function resetHostForTest(): void {
  resetHostInitCacheForTest();
  resetSocketRpcServersForTest();
  activeHandle = null;
  startPromise = null;
  appReadySignaled = false;
  hostRuntimeLayer = ForstNodeRuntimeLayer;
  hostLeaderOverrideForTest = null;
}

/** Test-only: bypass register preload check for direct startForstNodeHost calls. */
export function setHostLeaderOverrideForTest(value: boolean | null): void {
  hostLeaderOverrideForTest = value;
}

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
  const socketPath = options.socketPath ?? process.env[envSocketPath] ?? "";
  const deferAppReady = options.deferAppReady ?? false;

  if (!socketPath && process.platform !== "win32") {
    return yield* Effect.fail(HostErrors.hostSocketRequired());
  }

  const setup = createNodeRuntimeSetup(hostRuntimeLayer);
  const handle = yield* startSocketRpcServer({
    socketPath,
    readyPath,
    deferAppReady,
    exitProcessOnShutdown: false,
    runtime: setup.runtime,
    runtimeLayer: setup.layer,
    logPrefix: "host",
  });

  activeHandle = handle;

  return {
    socketPath: handle.socketPath,
    close: () => handle.close(),
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
  if (!activeHandle) {
    return yield* Effect.fail(HostErrors.hostReadyPathUnset());
  }
  if (!activeHandle.readyPath) {
    return yield* Effect.fail(HostErrors.hostReadyPathUnset());
  }
  if (!activeHandle.socketPath) {
    return yield* Effect.fail(HostErrors.hostSocketPathUnset());
  }

  yield* Effect.sync(() => {
    activeHandle!.signalReady();
    appReadySignaled = true;
  });
  yield* Effect.annotateCurrentSpan("socket", activeHandle.socketPath);
  yield* Effect.logInfo("host_app_ready").pipe(
    Effect.annotateLogs({
      event: "host_app_ready",
      socket: activeHandle.socketPath,
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

  startPromise = Effect.runPromise(
    hostStart(options).pipe(Effect.provide(setup.layer))
  );

  return Effect.tryPromise({
    try: () => startPromise!,
    catch: (cause) => causeToError(cause),
  });
}
