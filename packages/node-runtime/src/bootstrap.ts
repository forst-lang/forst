#!/usr/bin/env node
import { NodeRuntime } from "@effect/platform-node";
import { Effect } from "effect";
import { ForstNodeRuntimeLayer } from "./effect/layer.js";
import {
  defaultNodeRuntimeSetup,
  type ForstNodeRuntime,
} from "./effect/runtime.js";
import { startRpcServer } from "./rpc/server.js";

export interface BootstrapOptions {
  /** Runtime for proto-loop dispatch; must match the layer at the process boundary. */
  runtime?: ForstNodeRuntime;
}

export const bootstrapFatal = Effect.fn("Bootstrap.fatal")(function* (
  cause: unknown
) {
  const message = cause instanceof Error ? cause.message : String(cause);
  yield* Effect.annotateCurrentSpan("message", message);
  yield* Effect.logError("fatal").pipe(
    Effect.annotateLogs({ event: "fatal", message })
  );
  yield* Effect.sync(() => process.exit(1));
});

export const bootstrapMain = Effect.fn("Bootstrap.main")(function* (
  options: BootstrapOptions = {}
) {
  yield* Effect.annotateCurrentSpan("pid", process.pid);
  yield* Effect.logInfo("spawn").pipe(
    Effect.annotateLogs({ event: "spawn", pid: process.pid })
  );
  yield* startRpcServer(process.stdin, process.stdout, {
    exitProcessOnShutdown: true,
    runtime: options.runtime ?? defaultNodeRuntimeSetup.runtime,
  });
});

export function makeBootstrapProgram(
  options: BootstrapOptions = {}
): Effect.Effect<void, never, never> {
  return bootstrapMain(options).pipe(
    Effect.catchAllDefect((cause) => bootstrapFatal(cause))
  );
}

/** Default bootstrap program (stderr pretty logging via `ForstNodeRuntimeLayer`). */
export const bootstrapProgram = makeBootstrapProgram();

const isDirectExecution =
  typeof process.argv[1] === "string" &&
  (process.argv[1].endsWith("/bootstrap.js") ||
    process.argv[1].endsWith("/bootstrap.ts"));

if (isDirectExecution) {
  NodeRuntime.runMain(
    bootstrapProgram.pipe(Effect.provide(ForstNodeRuntimeLayer))
  );
}
