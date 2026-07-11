import { isClosedMethod, isStubMethod } from "../policy/methods.js";
import { handleAsyncCall, handleSyncCall } from "../runtime/calls.js";
import {
  handleGenClose,
  handleGenNext,
  handleGenNextBatch,
  handleGenOpen,
} from "../runtime/generators.js";
import {
  assertInitialized,
  createRuntimeState,
  initializeRuntime,
  shutdownRuntime,
  type RuntimeState,
} from "../runtime/lifecycle.js";
import { log } from "../logging/logger.js";
import {
  errorResponse,
  successResponse,
} from "./proto_loop.js";
import type { JsonRpcResponse } from "./protocol.js";
import {
  JsonRpcError,
  METHOD_NOT_FOUND,
  notImplemented,
  notInitialized,
} from "./errors.js";
import {
  METHOD_CALL,
  METHOD_CALL_ASYNC,
  METHOD_GEN_CLOSE,
  METHOD_GEN_NEXT,
  METHOD_GEN_NEXT_BATCH,
  METHOD_GEN_OPEN,
  METHOD_INITIALIZE,
  METHOD_PING,
  METHOD_SHUTDOWN,
  type CallParams,
  type GenCloseParams,
  type GenNextBatchParams,
  type GenNextParams,
  type GenOpenParams,
  type InitializeParams,
  type JsonRpcRequest,
} from "./protocol.js";

export interface DispatcherOptions {
  state?: RuntimeState;
}

export function createDispatcher(options: DispatcherOptions = {}) {
  const state = options.state ?? createRuntimeState();

  async function dispatch(request: JsonRpcRequest): Promise<JsonRpcResponse> {
    const id = request.id ?? null;
    const method = request.method;

    log("rpc_recv", {
      rpc_method: method,
      rpc_id: typeof id === "number" || typeof id === "string" ? id : null,
    });

    if (!isClosedMethod(method)) {
      log("policy_reject", { rpc_method: method, reason: "unknown_method" });
      return errorResponse(
        id,
        new JsonRpcError(METHOD_NOT_FOUND, "Method not found")
      );
    }

    if (isStubMethod(method)) {
      log("policy_reject", { rpc_method: method, reason: "not_implemented" });
      return errorResponse(id, notImplemented(method));
    }

    if (method !== METHOD_INITIALIZE && !state.initialized) {
      log("policy_reject", { rpc_method: method, reason: "not_initialized" });
      return errorResponse(id, notInitialized());
    }

    try {
      const result = await handleMethod(state, method, request.params);
      log("rpc_send", {
        rpc_method: method,
        rpc_id: typeof id === "number" || typeof id === "string" ? id : null,
        ok: true,
      });
      return successResponse(id, result);
    } catch (err) {
      const rpcErr =
        err instanceof JsonRpcError
          ? err
          : new JsonRpcError(METHOD_NOT_FOUND, String(err));

      log("rpc_send", {
        rpc_method: method,
        rpc_id: typeof id === "number" || typeof id === "string" ? id : null,
        ok: false,
        error_code: rpcErr.code,
      });

      return errorResponse(id, rpcErr);
    }
  }

  return { dispatch, state };
}

async function handleMethod(
  state: RuntimeState,
  method: string,
  params: unknown
): Promise<unknown> {
  switch (method) {
    case METHOD_INITIALIZE:
      return initializeRuntime(state, params as InitializeParams);
    case METHOD_PING:
      return { pong: true as const };
    case METHOD_CALL: {
      const index = assertInitialized(state);
      return handleSyncCall(index, params as CallParams);
    }
    case METHOD_CALL_ASYNC: {
      const index = assertInitialized(state);
      return handleAsyncCall(index, params as CallParams);
    }
    case METHOD_GEN_OPEN: {
      const index = assertInitialized(state);
      return handleGenOpen(index, params as GenOpenParams);
    }
    case METHOD_GEN_NEXT:
      return handleGenNext(params as GenNextParams);
    case METHOD_GEN_NEXT_BATCH:
      return handleGenNextBatch(params as GenNextBatchParams);
    case METHOD_GEN_CLOSE:
      return handleGenClose(params as GenCloseParams);
    case METHOD_SHUTDOWN:
      return shutdownRuntime(state);
    default:
      throw new JsonRpcError(METHOD_NOT_FOUND, "Method not found");
  }
}

export type Dispatcher = ReturnType<typeof createDispatcher>;
