import type { ManifestIndex } from "../policy/manifest.js";
import { assertExportAllowed } from "../policy/manifest.js";
import {
  modulePathToFileUrl,
  resolveModulePath,
} from "../policy/paths.js";
import { applicationError } from "../rpc/errors.js";
import type { CallParams, CallResult } from "../rpc/protocol.js";
import { log } from "../logging/logger.js";
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

/**
 * Async call handler — awaits the returned Promise before responding.
 * Works with ordinary `export async function` TypeScript exports.
 */
export async function handleAsyncCall(
  index: ManifestIndex,
  params: CallParams
): Promise<CallResult> {
  if (params === null || typeof params !== "object") {
    throw applicationError("call params must be an object");
  }

  const { moduleId, exportName } = params;
  if (typeof moduleId !== "string" || typeof exportName !== "string") {
    throw applicationError("call requires moduleId and exportName strings");
  }

  assertExportAllowed(index, moduleId, exportName, "asyncFunction");

  const absPath = await resolveModulePath(index.boundaryRoot, moduleId);
  const fileUrl = modulePathToFileUrl(absPath);
  const mod = await importModule(fileUrl);
  const fn = getExportFunction(mod, exportName);
  const args = Array.isArray(params.args) ? params.args : [];

  log("callAsync", {
    module_id: moduleId,
    export_name: exportName,
    arg_count: args.length,
  });

  try {
    const value = await fn(...args);
    return { value };
  } catch (err) {
    throw serializeThrownError(err, moduleId, exportName);
  }
}

/**
 * Synchronous call handler — does NOT await the return value.
 * Works with ordinary `export function` TypeScript exports.
 */
export async function handleSyncCall(
  index: ManifestIndex,
  params: CallParams
): Promise<CallResult> {
  if (params === null || typeof params !== "object") {
    throw applicationError("call params must be an object");
  }

  const { moduleId, exportName } = params;
  if (typeof moduleId !== "string" || typeof exportName !== "string") {
    throw applicationError("call requires moduleId and exportName strings");
  }

  assertExportAllowed(index, moduleId, exportName, "function");

  const absPath = await resolveModulePath(index.boundaryRoot, moduleId);
  const fileUrl = modulePathToFileUrl(absPath);
  const mod = await importModule(fileUrl);
  const fn = getExportFunction(mod, exportName);
  const args = Array.isArray(params.args) ? params.args : [];

  log("call", {
    module_id: moduleId,
    export_name: exportName,
    arg_count: args.length,
  });

  try {
    const value = fn(...args);
    return { value };
  } catch (err) {
    throw serializeThrownError(err, moduleId, exportName);
  }
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
