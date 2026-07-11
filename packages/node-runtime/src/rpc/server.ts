import type { Writable } from "node:stream";
import { Effect } from "effect";
import {
  defaultNodeRuntimeSetup,
  type ForstNodeRuntime,
} from "../effect/runtime.js";
import { createDispatcher, type Dispatcher } from "./dispatcher.js";
import { runProtoLoop } from "./proto_loop.js";
import { METHOD_SHUTDOWN } from "./protocol.js";
import type { JsonRpcRequest } from "./protocol.js";
import type { RuntimeState } from "../runtime/lifecycle.js";

export interface RpcServerOptions {
  /** When true (bootstrap child), exit the process after shutdown RPC completes. */
  exitProcessOnShutdown?: boolean;
  /** Runtime for proto-loop dispatch; must match the layer at the process boundary. */
  runtime?: ForstNodeRuntime;
}

const onRequest = Effect.fn("Rpc.onRequest")(
  function* (
    request: JsonRpcRequest,
    dispatch: Dispatcher["dispatch"],
    state: RuntimeState,
    exitOnShutdown: boolean
  ) {
    yield* Effect.annotateCurrentSpan("rpc_method", request.method);
    const response = yield* dispatch(request);
    if (request.method === METHOD_SHUTDOWN && state.shuttingDown) {
      if (exitOnShutdown) {
        yield* Effect.sync(() => {
          setImmediate(() => process.exit(0));
        });
      }
    }
    return response;
  }
);

/** Runs the Forst Node RPC loop on the given streams. */
export const startRpcServer = Effect.fn("Rpc.startServer")(
  function* (
    input: NodeJS.ReadableStream,
    output: Writable,
    options: RpcServerOptions = {}
  ) {
    const exitOnShutdown = options.exitProcessOnShutdown ?? false;

    yield* Effect.annotateCurrentSpan("exit_on_shutdown", exitOnShutdown);
    yield* Effect.annotateCurrentSpan("pid", process.pid);

    yield* Effect.logInfo("rpc_server_start").pipe(
      Effect.annotateLogs({
        event: "rpc_server_start",
        pid: process.pid,
        exit_on_shutdown: exitOnShutdown,
      })
    );

    const { dispatch, state } = createDispatcher();
    yield* runProtoLoop(input, output, {
      runtime: options.runtime ?? defaultNodeRuntimeSetup.runtime,
      onRequest: (request) =>
        onRequest(request, dispatch, state, exitOnShutdown),
    });

    yield* Effect.logInfo("rpc_server_closed").pipe(
      Effect.annotateLogs({ event: "rpc_server_closed" })
    );
    if (exitOnShutdown) {
      yield* Effect.sync(() => process.exit(0));
    }
  }
);
