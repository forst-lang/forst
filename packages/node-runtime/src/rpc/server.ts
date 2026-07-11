import type { Writable } from "node:stream";
import { createDispatcher } from "./dispatcher.js";
import { runProtoLoop } from "./proto_loop.js";
import { log } from "../logging/logger.js";
import { METHOD_SHUTDOWN } from "./protocol.js";

export interface RpcServerOptions {
  /** When true (bootstrap child), exit the process after shutdown RPC completes. */
  exitProcessOnShutdown?: boolean;
}

/**
 * Runs the Forst Node RPC loop on the given streams.
 * Invariants: one client per server; logs go to stderr only; manifest allowlist enforced by dispatcher.
 */
export async function startRpcServer(
  input: NodeJS.ReadableStream,
  output: Writable,
  options: RpcServerOptions = {}
): Promise<void> {
  const exitOnShutdown = options.exitProcessOnShutdown ?? false;

  log("rpc_server_start", { pid: process.pid, exit_on_shutdown: exitOnShutdown });

  const { dispatch, state } = createDispatcher();
  const loopOptions = {
    onRequest: async (request: Parameters<typeof dispatch>[0]) => {
      const response = await dispatch(request);
      if (request.method === METHOD_SHUTDOWN && state.shuttingDown) {
        if (exitOnShutdown) {
          setImmediate(() => process.exit(0));
        }
      }
      return response;
    },
  };

  await runProtoLoop(input, output, loopOptions);

  log("rpc_server_closed", {});
  if (exitOnShutdown) {
    process.exit(0);
  }
}
