import type { ManifestIndex } from "../policy/manifest.js";
import { assertExportAllowed } from "../policy/manifest.js";
import {
  modulePathToFileUrl,
  resolveModulePath,
} from "../policy/paths.js";
import { applicationError } from "../rpc/errors.js";
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
import { log } from "../logging/logger.js";
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
    throw applicationError(`export is not a function: ${exportName}`, {
      exportName,
    });
  }
  return value as (...args: unknown[]) => unknown;
}

async function loadGeneratorExport(
  index: ManifestIndex,
  params: CallParams
): Promise<{ fn: (...args: unknown[]) => unknown; isAsync: boolean }> {
  if (params === null || typeof params !== "object") {
    throw applicationError("gen params must be an object");
  }

  const { moduleId, exportName } = params;
  if (typeof moduleId !== "string" || typeof exportName !== "string") {
    throw applicationError("gen requires moduleId and exportName strings");
  }

  const entry = assertExportAllowed(index, moduleId, exportName);
  if (entry.kind !== "generator" && entry.kind !== "asyncGenerator") {
    throw applicationError("export is not a generator", {
      moduleId,
      exportName,
      kind: entry.kind,
    });
  }

  const absPath = await resolveModulePath(index.boundaryRoot, moduleId);
  const fileUrl = modulePathToFileUrl(absPath);
  const mod = await importModule(fileUrl);
  return {
    fn: getExportFunction(mod, exportName),
    isAsync: entry.kind === "asyncGenerator",
  };
}

function serializeThrownError(
  err: unknown,
  moduleId: string,
  exportName: string
): ReturnType<typeof applicationError> {
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

function requireStream(streamId: string): StreamState {
  const stream = streams.get(streamId);
  if (!stream || stream.closed) {
    throw applicationError("invalid or closed streamId", { streamId });
  }
  return stream;
}

export async function handleGenOpen(
  index: ManifestIndex,
  params: GenOpenParams
): Promise<GenOpenResult> {
  const { fn, isAsync } = await loadGeneratorExport(index, params);
  const args = Array.isArray(params.args) ? params.args : [];

  log("gen_open", {
    module_id: params.moduleId,
    export_name: params.exportName,
    arg_count: args.length,
    async: isAsync,
  });

  let iterator: AnyIterator;
  try {
    const value = fn(...args);
    if (
      value === null ||
      typeof value !== "object" ||
      typeof (value as AsyncIterator<unknown>).next !== "function"
    ) {
      throw applicationError("export did not return an iterator", {
        moduleId: params.moduleId,
        exportName: params.exportName,
      });
    }
    iterator = value as AnyIterator;
  } catch (err) {
    throw serializeThrownError(err, params.moduleId, params.exportName);
  }

  const streamId = String(nextStreamId++);
  streams.set(streamId, { iterator, isAsync, closed: false });
  openStreamCount += 1;

  return { streamId };
}

export async function handleGenNext(
  params: GenNextParams
): Promise<GenNextResult> {
  if (params === null || typeof params !== "object") {
    throw applicationError("genNext params must be an object");
  }
  const { streamId } = params;
  if (typeof streamId !== "string" || streamId === "") {
    throw applicationError("genNext requires streamId string");
  }

  const stream = requireStream(streamId);

  log("gen_next", { stream_id: streamId, async: stream.isAsync });

  try {
    const result = stream.isAsync
      ? await (stream.iterator as AsyncIterator<unknown>).next()
      : (stream.iterator as SyncIterator).next();
    return stepFromIteratorResult(result);
  } catch (err) {
    return {
      kind: "error",
      message: err instanceof Error ? err.message : String(err),
      data:
        err instanceof Error
          ? { name: err.name, stack: err.stack }
          : undefined,
    };
  }
}

export async function handleGenNextBatch(
  params: GenNextBatchParams
): Promise<GenNextBatchResult> {
  if (params === null || typeof params !== "object") {
    throw applicationError("genNextBatch params must be an object");
  }
  const { streamId } = params;
  if (typeof streamId !== "string" || streamId === "") {
    throw applicationError("genNextBatch requires streamId string");
  }

  let maxItems = params.maxItems ?? 1;
  if (typeof maxItems !== "number" || !Number.isInteger(maxItems) || maxItems <= 0) {
    maxItems = 1;
  }

  log("gen_next_batch", { stream_id: streamId, max_items: maxItems });

  const steps: GenNextResult[] = [];
  for (let i = 0; i < maxItems; i++) {
    const step = await handleGenNext({ streamId });
    steps.push(step);
    if (step.kind === "done" || step.kind === "error") {
      break;
    }
  }
  return { steps };
}

export async function handleGenClose(
  params: GenCloseParams
): Promise<GenCloseResult> {
  if (params === null || typeof params !== "object") {
    throw applicationError("genClose params must be an object");
  }
  const { streamId } = params;
  if (typeof streamId !== "string" || streamId === "") {
    throw applicationError("genClose requires streamId string");
  }

  const stream = streams.get(streamId);
  if (!stream || stream.closed) {
    return { ok: true };
  }

  stream.closed = true;
  streams.delete(streamId);
  openStreamCount = Math.max(0, openStreamCount - 1);

  log("gen_close", { stream_id: streamId, async: stream.isAsync });

  const iterator = stream.iterator as {
    return?: (value?: unknown) => unknown;
  };
  if (typeof iterator.return === "function") {
    try {
      if (stream.isAsync) {
        await iterator.return();
      } else {
        iterator.return();
      }
    } catch {
      // Idempotent cleanup — ignore return() failures.
    }
  }

  return { ok: true };
}
