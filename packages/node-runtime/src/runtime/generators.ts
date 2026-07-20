import { Effect } from "effect";
import type { ManifestIndex } from "../policy/manifest.js";
import { assertExportAllowedEffect } from "../policy/export_allowed.js";
import {
  modulePathToFileUrl,
  resolveModulePath,
} from "../policy/paths.js";
import * as Errors from "../rpc/errors.js";
import type {
  CallParams,
  GenCloseParams,
  GenCloseResult,
  GenNextBatchParams,
  GenNextBatchResult,
  GenNextParams,
  GenNextResult,
  GenOpenParams,
  GenOpenResult,
} from "../rpc/protocol.js";
import { importModule } from "./module_cache.js";
import { resolveExportValue } from "./export_value.js";

type SyncIterator = Iterator<unknown, unknown, unknown>;
type AnyIterator = SyncIterator | AsyncIterator<unknown, unknown, unknown>;

interface StreamState {
  iterator: AnyIterator;
  isAsync: boolean;
  closed: boolean;
}

const streams = new Map<string, StreamState>();
let nextStreamId = 1;

/** Tracks open streams for leak-detection tests. */
export let openStreamCount = 0;

/** Test-only cleanup so stream maps and leak counters do not leak across test cases. */
export function resetGeneratorStateForTest(): void {
  streams.clear();
  nextStreamId = 1;
  openStreamCount = 0;
}

function getExportFunction(
  mod: Record<string, unknown>,
  exportName: string
): (...args: unknown[]) => unknown {
  const value = resolveExportValue(mod, exportName);
  if (typeof value !== "function") {
    throw Errors.applicationError(`export is not a function: ${exportName}`, {
      exportName,
    });
  }
  return value as (...args: unknown[]) => unknown;
}

function serializeThrownError(
  err: unknown,
  moduleId: string,
  exportName: string
): Errors.JsonRpcError {
  if (err instanceof Errors.JsonRpcError) {
    return err;
  }
  if (err instanceof Error) {
    return Errors.applicationError(err.message, {
      name: err.name,
      stack: err.stack,
      moduleId,
      exportName,
    });
  }
  return Errors.applicationError(String(err), { moduleId, exportName });
}

const loadGeneratorExport = Effect.fn("Runtime.loadGeneratorExport")(
  function* (index: ManifestIndex, params: CallParams) {
    if (params === null || typeof params !== "object") {
      return yield* Effect.fail(Errors.applicationError("gen params must be an object"));
    }

    const { moduleId, exportName } = params;
    if (typeof moduleId !== "string" || typeof exportName !== "string") {
      return yield* Effect.fail(
        Errors.applicationError("gen requires moduleId and exportName strings")
      );
    }

    const entry = yield* assertExportAllowedEffect(
      index,
      moduleId,
      exportName
    );
    if (entry.kind !== "generator" && entry.kind !== "asyncGenerator") {
      return yield* Effect.fail(
        Errors.applicationError("export is not a generator", {
          moduleId,
          exportName,
          kind: entry.kind,
        })
      );
    }

    const absPath = yield* Effect.tryPromise({
      try: () => resolveModulePath(index.boundaryRoot, moduleId),
      catch: (cause) => serializeThrownError(cause, moduleId, exportName),
    });
    const fileUrl = modulePathToFileUrl(absPath);
    const mod = yield* importModule(fileUrl).pipe(
      Effect.mapError((cause) => serializeThrownError(cause, moduleId, exportName))
    );
    const fn = yield* Effect.try({
      try: () => getExportFunction(mod, exportName),
      catch: (cause) => serializeThrownError(cause, moduleId, exportName),
    });
    return {
      fn,
      isAsync: entry.kind === "asyncGenerator",
    };
  }
);

function stepFromIteratorResult(
  result: IteratorResult<unknown, unknown>
): GenNextResult {
  if (result.done) {
    const step: GenNextResult = { kind: "done" };
    if (result.value !== undefined) {
      step.value = result.value;
    }
    return step;
  }
  return { kind: "yield", value: result.value };
}

const requireStream = Effect.fn("Runtime.requireStream")(function* (
  streamId: string
) {
  yield* Effect.annotateCurrentSpan("stream_id", streamId);
  const stream = streams.get(streamId);
  if (!stream || stream.closed) {
    return yield* Effect.fail(
      Errors.applicationError("invalid or closed streamId", { streamId })
    );
  }
  return stream;
});

/** `genOpen` RPC handler: loads an allowlisted generator export and registers a server-side stream id for later steps. */
export const handleGenOpen = Effect.fn("Runtime.handleGenOpen")(
  function* (index: ManifestIndex, params: GenOpenParams) {
    const { fn, isAsync } = yield* loadGeneratorExport(index, params);
    const args = Array.isArray(params.args) ? params.args : [];

    yield* Effect.annotateCurrentSpan("module_id", params.moduleId);
    yield* Effect.annotateCurrentSpan("export_name", params.exportName);
    yield* Effect.annotateCurrentSpan("arg_count", args.length);
    yield* Effect.annotateCurrentSpan("async", isAsync);

    yield* Effect.logDebug("gen_open").pipe(
      Effect.annotateLogs({
        event: "gen_open",
        module_id: params.moduleId,
        export_name: params.exportName,
        arg_count: args.length,
        async: isAsync,
      })
    );

    const iterator = yield* Effect.try({
      try: () => {
        const value = fn(...args);
        if (
          value === null ||
          typeof value !== "object" ||
          typeof (value as AsyncIterator<unknown>).next !== "function"
        ) {
          throw Errors.applicationError("export did not return an iterator", {
            moduleId: params.moduleId,
            exportName: params.exportName,
          });
        }
        return value as AnyIterator;
      },
      catch: (cause) =>
        serializeThrownError(cause, params.moduleId, params.exportName),
    });

    const streamId = String(nextStreamId++);
    streams.set(streamId, { iterator, isAsync, closed: false });
    openStreamCount += 1;

    return { streamId } satisfies GenOpenResult;
  }
);

/** `genNext` RPC handler: advances one iterator step and serializes yield, done, or error for the client. */
export const handleGenNext = Effect.fn("Runtime.handleGenNext")(
  function* (params: GenNextParams) {
    if (params === null || typeof params !== "object") {
      return yield* Effect.fail(
        Errors.applicationError("genNext params must be an object")
      );
    }
    const { streamId } = params;
    if (typeof streamId !== "string" || streamId === "") {
      return yield* Effect.fail(
        Errors.applicationError("genNext requires streamId string")
      );
    }

    const stream = yield* requireStream(streamId);

    yield* Effect.annotateCurrentSpan("async", stream.isAsync);

    yield* Effect.logDebug("gen_next").pipe(
      Effect.annotateLogs({
        event: "gen_next",
        stream_id: streamId,
        async: stream.isAsync,
      })
    );

    return yield* Effect.tryPromise({
      try: async () => {
        try {
          const result = stream.isAsync
            ? await (stream.iterator as AsyncIterator<unknown>).next()
            : (stream.iterator as SyncIterator).next();
          return stepFromIteratorResult(result);
        } catch (err) {
          return {
            kind: "error" as const,
            message: err instanceof Error ? err.message : String(err),
            data:
              err instanceof Error
                ? { name: err.name, stack: err.stack }
                : undefined,
          };
        }
      },
      catch: (cause) =>
        Errors.applicationError(cause instanceof Error ? cause.message : String(cause)),
    });
  }
);

/** Batch `genNext` for fewer round trips when the client wants multiple yields per RPC. */
export const handleGenNextBatch = Effect.fn("Runtime.handleGenNextBatch")(
  function* (params: GenNextBatchParams) {
    if (params === null || typeof params !== "object") {
      return yield* Effect.fail(
        Errors.applicationError("genNextBatch params must be an object")
      );
    }
    const { streamId } = params;
    if (typeof streamId !== "string" || streamId === "") {
      return yield* Effect.fail(
        Errors.applicationError("genNextBatch requires streamId string")
      );
    }

    let maxItems = params.maxItems ?? 1;
    if (
      typeof maxItems !== "number" ||
      !Number.isInteger(maxItems) ||
      maxItems <= 0
    ) {
      maxItems = 1;
    }

    yield* Effect.annotateCurrentSpan("stream_id", streamId);
    yield* Effect.annotateCurrentSpan("max_items", maxItems);

    yield* Effect.logDebug("gen_next_batch").pipe(
      Effect.annotateLogs({
        event: "gen_next_batch",
        stream_id: streamId,
        max_items: maxItems,
      })
    );

    const steps: GenNextResult[] = [];
    for (let i = 0; i < maxItems; i++) {
      const step = yield* handleGenNext({ streamId });
      steps.push(step);
      if (step.kind === "done" || step.kind === "error") {
        break;
      }
    }
    return { steps } satisfies GenNextBatchResult;
  }
);

/** `genClose` RPC handler: tears down the stream and invokes `iterator.return()` so generator cleanup runs on the host. */
export const handleGenClose = Effect.fn("Runtime.handleGenClose")(
  function* (params: GenCloseParams) {
    if (params === null || typeof params !== "object") {
      return yield* Effect.fail(
        Errors.applicationError("genClose params must be an object")
      );
    }
    const { streamId } = params;
    if (typeof streamId !== "string" || streamId === "") {
      return yield* Effect.fail(
        Errors.applicationError("genClose requires streamId string")
      );
    }

    const stream = streams.get(streamId);
    if (!stream || stream.closed) {
      return { ok: true as const };
    }

    stream.closed = true;
    streams.delete(streamId);
    openStreamCount = Math.max(0, openStreamCount - 1);

    yield* Effect.annotateCurrentSpan("stream_id", streamId);
    yield* Effect.annotateCurrentSpan("async", stream.isAsync);

    yield* Effect.logDebug("gen_close").pipe(
      Effect.annotateLogs({
        event: "gen_close",
        stream_id: streamId,
        async: stream.isAsync,
      })
    );

    const iterator = stream.iterator as {
      return?: (value?: unknown) => unknown;
    };
    if (typeof iterator.return === "function") {
      yield* Effect.tryPromise({
        try: async () => {
          if (stream.isAsync) {
            await iterator.return!();
          } else {
            iterator.return!();
          }
        },
        catch: () => undefined,
      }).pipe(Effect.ignore);
    }

    return { ok: true as const } satisfies GenCloseResult;
  }
);
