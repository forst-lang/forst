#!/usr/bin/env node
/**
 * Bootstrap child process entrypoint for the Forst Node RPC server.
 *
 * Reads `FORST_NODE_SOCKET` / `FORST_NODE_READY`, starts the socket RPC server,
 * and runs until shutdown. Used by the Go sidecar as a managed child process.
 *
 * @module forst_node_bootstrap
 */
import { NodeRuntime } from "@effect/platform-node";
import { Effect } from "effect";
import { ForstNodeRuntimeLayer } from "./effect/layer.js";
import { createNodeRuntimeSetup } from "./effect/runtime.js";
import {
  envReadyPath,
  envSocketPath,
  isWindows,
  startSocketRpcServer,
} from "./rpc/socket_server.js";

/** Options passed to bootstrap Effect programs. */
export interface BootstrapOptions {
  /** Runtime for proto-loop dispatch; must match the layer at the process boundary. */
  runtime?: import("./effect/runtime.js").ForstNodeRuntime;
}

/** Logs a fatal error and exits the process with code 1. */
export const bootstrapFatal: (cause: unknown) => Effect.Effect<void, never, never> =
  Effect.fn("Bootstrap.fatal")(function* (cause: unknown) {
  const message = cause instanceof Error ? cause.message : String(cause);
  const stack =
    cause instanceof Error && cause.stack !== undefined ? cause.stack : undefined;
  yield* Effect.annotateCurrentSpan("message", message);
  if (stack !== undefined) {
    yield* Effect.annotateCurrentSpan("stack", stack);
  }
  const logFields: Record<string, string> = { event: "fatal", message };
  if (stack !== undefined) {
    logFields.stack = stack;
  }
  yield* Effect.logError("fatal").pipe(Effect.annotateLogs(logFields));
  yield* Effect.sync(() => process.exit(1));
});

/** Starts the bootstrap socket RPC server and runs until shutdown. */
export const bootstrapMain: (
  options?: BootstrapOptions
) => Effect.Effect<void, Error, never> = Effect.fn("Bootstrap.main")(function* (
  options: BootstrapOptions = {}
) {
  yield* Effect.annotateCurrentSpan("pid", process.pid);
  yield* Effect.logInfo("spawn").pipe(
    Effect.annotateLogs({ event: "spawn", pid: process.pid })
  );

  const socketPath = process.env[envSocketPath] ?? "";
  const readyPath = process.env[envReadyPath] ?? "";
  if (!socketPath && !isWindows()) {
    return yield* bootstrapFatal(
      new Error("FORST_NODE_SOCKET is required on Unix")
    );
  }

  const setup = createNodeRuntimeSetup(ForstNodeRuntimeLayer);
  const runtime = options.runtime ?? setup.runtime;

  yield* startSocketRpcServer({
    socketPath,
    readyPath,
    deferAppReady: false,
    exitProcessOnShutdown: true,
    runtime,
    runtimeLayer: setup.layer,
    logPrefix: "bootstrap",
  }).pipe(
    Effect.flatMap(() => Effect.never),
    Effect.provide(setup.layer)
  );
});

/** Builds a bootstrap program with fatal error handling. */
export function makeBootstrapProgram(
  options: BootstrapOptions = {}
): Effect.Effect<void, never, never> {
  return bootstrapMain(options).pipe(
    Effect.catchAll((err) => bootstrapFatal(err)),
    Effect.catchAllDefect((cause) => bootstrapFatal(cause))
  );
}

/** Default bootstrap program (stderr pretty logging via `ForstNodeRuntimeLayer`). */
export const bootstrapProgram: Effect.Effect<void, never, never> =
  makeBootstrapProgram();

const isDirectExecution =
  typeof process.argv[1] === "string" &&
  (process.argv[1].endsWith("/bootstrap.js") ||
    process.argv[1].endsWith("/bootstrap.ts"));

if (isDirectExecution) {
  NodeRuntime.runMain(
    bootstrapProgram.pipe(Effect.provide(ForstNodeRuntimeLayer)),
    { disablePrettyLogger: true }
  );
}
