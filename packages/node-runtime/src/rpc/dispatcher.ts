import { Effect, Either } from "effect";
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

const handleMethod = Effect.fn("Rpc.handleMethod")(
  function* (state: RuntimeState, method: string, params: unknown) {
    yield* Effect.annotateCurrentSpan("rpc_method", method);
    switch (method) {
      case METHOD_INITIALIZE:
        return yield* initializeRuntime(state, params as InitializeParams);
      case METHOD_PING:
        return { pong: true as const };
      case METHOD_CALL: {
        const index = assertInitialized(state);
        return yield* handleSyncCall(index, params as CallParams);
      }
      case METHOD_CALL_ASYNC: {
        const index = assertInitialized(state);
        return yield* handleAsyncCall(index, params as CallParams);
      }
      case METHOD_GEN_OPEN: {
        const index = assertInitialized(state);
        return yield* handleGenOpen(index, params as GenOpenParams);
      }
      case METHOD_GEN_NEXT:
        return yield* handleGenNext(params as GenNextParams);
      case METHOD_GEN_NEXT_BATCH:
        return yield* handleGenNextBatch(params as GenNextBatchParams);
      case METHOD_GEN_CLOSE:
        return yield* handleGenClose(params as GenCloseParams);
      case METHOD_SHUTDOWN:
        return yield* shutdownRuntime(state);
      default:
        return yield* Effect.fail(
          new JsonRpcError(METHOD_NOT_FOUND, "Method not found")
        );
    }
  }
);

function rpcIdSpanValue(id: JsonRpcRequest["id"]): number | string | null {
  return typeof id === "number" || typeof id === "string" ? id : null;
}

export function createDispatcher(options: DispatcherOptions = {}) {
  const state = options.state ?? createRuntimeState();

  const dispatch = Effect.fn("Rpc.dispatch")(function* (request: JsonRpcRequest) {
    const id = request.id ?? null;
    const method = request.method;
    const rpcId = rpcIdSpanValue(id);

    yield* Effect.annotateCurrentSpan("rpc_method", method);
    yield* Effect.annotateCurrentSpan("rpc_id", rpcId);

    yield* Effect.logDebug("rpc_recv").pipe(
      Effect.annotateLogs({
        event: "rpc_recv",
        rpc_method: method,
        rpc_id: rpcId,
      })
    );

    if (!isClosedMethod(method)) {
      yield* Effect.annotateCurrentSpan("reason", "unknown_method");
      yield* Effect.logWarning("policy_reject").pipe(
        Effect.annotateLogs({
          event: "policy_reject",
          rpc_method: method,
          reason: "unknown_method",
        })
      );
      return errorResponse(
        id,
        new JsonRpcError(METHOD_NOT_FOUND, "Method not found")
      );
    }

    if (isStubMethod(method)) {
      yield* Effect.annotateCurrentSpan("reason", "not_implemented");
      yield* Effect.logWarning("policy_reject").pipe(
        Effect.annotateLogs({
          event: "policy_reject",
          rpc_method: method,
          reason: "not_implemented",
        })
      );
      return errorResponse(id, notImplemented(method));
    }

    if (method !== METHOD_INITIALIZE && !state.initialized) {
      yield* Effect.annotateCurrentSpan("reason", "not_initialized");
      yield* Effect.logWarning("policy_reject").pipe(
        Effect.annotateLogs({
          event: "policy_reject",
          rpc_method: method,
          reason: "not_initialized",
        })
      );
      return errorResponse(id, notInitialized());
    }

    const outcome = yield* handleMethod(state, method, request.params).pipe(
      Effect.either
    );

    if (Either.isLeft(outcome)) {
      const rpcErr = outcome.left;
      yield* Effect.annotateCurrentSpan("ok", false);
      yield* Effect.annotateCurrentSpan("error_code", rpcErr.code);
      yield* Effect.logDebug("rpc_send").pipe(
        Effect.annotateLogs({
          event: "rpc_send",
          rpc_method: method,
          rpc_id: rpcId,
          ok: false,
          error_code: rpcErr.code,
        })
      );
      return errorResponse(id, rpcErr);
    }

    yield* Effect.annotateCurrentSpan("ok", true);
    yield* Effect.logDebug("rpc_send").pipe(
      Effect.annotateLogs({
        event: "rpc_send",
        rpc_method: method,
        rpc_id: rpcId,
        ok: true,
      })
    );
    return successResponse(id, outcome.right);
  });

  return { dispatch, state };
}

export type Dispatcher = ReturnType<typeof createDispatcher>;
