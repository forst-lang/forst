import { Effect } from "effect";
import type { ManifestIndex } from "../policy/manifest.js";
import { assertExportAllowedEffect } from "../policy/export_allowed.js";
import {
  modulePathToFileUrl,
  resolveModulePath,
} from "../policy/paths.js";
import { applicationError, type JsonRpcError } from "../rpc/errors.js";
import type { CallParams, CallResult } from "../rpc/protocol.js";
import { importModule } from "./module_cache.js";
import { resolveExportValue } from "./export_value.js";

function getExportFunction(
  mod: Record<string, unknown>,
  exportName: string
): (...args: unknown[]) => unknown {
  const value = resolveExportValue(mod, exportName);
  if (typeof value !== "function") {
    throw applicationError(`export is not a function: ${exportName}`, {
      exportName,
    });
  }
  return value as (...args: unknown[]) => unknown;
}

function serializeThrownError(
  err: unknown,
  moduleId: string,
  exportName: string
): JsonRpcError {
  if (err instanceof Error && "code" in err) {
    return err as JsonRpcError;
  }
  if (err instanceof Error) {
    return applicationError(err.message, {
      name: err.name,
      stack: err.stack,
      moduleId,
      exportName,
    });
  }
  return applicationError(String(err), { moduleId, exportName });
}

/** Async call handler — awaits the returned Promise before responding. */
export const handleAsyncCall = Effect.fn("Runtime.handleAsyncCall")(
  function* (index: ManifestIndex, params: CallParams) {
    if (params === null || typeof params !== "object") {
      return yield* Effect.fail(applicationError("call params must be an object"));
    }

    const { moduleId, exportName } = params;
    if (typeof moduleId !== "string" || typeof exportName !== "string") {
      return yield* Effect.fail(
        applicationError("call requires moduleId and exportName strings")
      );
    }

    yield* assertExportAllowedEffect(
      index,
      moduleId,
      exportName,
      "asyncFunction"
    );

    const absPath = yield* Effect.tryPromise({
      try: () => resolveModulePath(index.boundaryRoot, moduleId),
      catch: (cause) => serializeThrownError(cause, moduleId, exportName),
    });
    const fileUrl = modulePathToFileUrl(absPath);
    const mod = yield* importModule(fileUrl).pipe(
      Effect.mapError((cause) => serializeThrownError(cause, moduleId, exportName))
    );
    const fn = getExportFunction(mod, exportName);
    const args = Array.isArray(params.args) ? params.args : [];

    yield* Effect.annotateCurrentSpan("module_id", moduleId);
    yield* Effect.annotateCurrentSpan("export_name", exportName);
    yield* Effect.annotateCurrentSpan("arg_count", args.length);

    yield* Effect.logDebug("callAsync").pipe(
      Effect.annotateLogs({
        event: "callAsync",
        module_id: moduleId,
        export_name: exportName,
        arg_count: args.length,
      })
    );

    const value = yield* Effect.tryPromise({
      try: () => Promise.resolve(fn(...args)),
      catch: (cause) => serializeThrownError(cause, moduleId, exportName),
    });
    return { value } satisfies CallResult;
  }
);

/** Synchronous call handler — does NOT await the return value. */
export const handleSyncCall = Effect.fn("Runtime.handleSyncCall")(
  function* (index: ManifestIndex, params: CallParams) {
    if (params === null || typeof params !== "object") {
      return yield* Effect.fail(applicationError("call params must be an object"));
    }

    const { moduleId, exportName } = params;
    if (typeof moduleId !== "string" || typeof exportName !== "string") {
      return yield* Effect.fail(
        applicationError("call requires moduleId and exportName strings")
      );
    }

    yield* assertExportAllowedEffect(index, moduleId, exportName, "function");

    const absPath = yield* Effect.tryPromise({
      try: () => resolveModulePath(index.boundaryRoot, moduleId),
      catch: (cause) => serializeThrownError(cause, moduleId, exportName),
    });
    const fileUrl = modulePathToFileUrl(absPath);
    const mod = yield* importModule(fileUrl).pipe(
      Effect.mapError((cause) => serializeThrownError(cause, moduleId, exportName))
    );
    const fn = getExportFunction(mod, exportName);
    const args = Array.isArray(params.args) ? params.args : [];

    yield* Effect.annotateCurrentSpan("module_id", moduleId);
    yield* Effect.annotateCurrentSpan("export_name", exportName);
    yield* Effect.annotateCurrentSpan("arg_count", args.length);

    yield* Effect.logDebug("call").pipe(
      Effect.annotateLogs({
        event: "call",
        module_id: moduleId,
        export_name: exportName,
        arg_count: args.length,
      })
    );

    const value = yield* Effect.try({
      try: () => fn(...args),
      catch: (cause) => serializeThrownError(cause, moduleId, exportName),
    });
    return { value } satisfies CallResult;
  }
);
