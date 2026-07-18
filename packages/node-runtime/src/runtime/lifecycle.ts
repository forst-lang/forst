import { Effect } from "effect";
import {
  buildManifestIndex,
  validateManifest,
  type ManifestIndex,
} from "../policy/manifest.js";
import { setFilesExcludePatterns } from "../policy/paths.js";
import * as Errors from "../rpc/errors.js";
import {
  PROTOCOL_VERSION,
  WIRE_PROTOCOL_PROTO_V1,
  type InitializeParams,
  type InitializeResult,
} from "../rpc/protocol.js";
import type { ForstNodeManifestV1 } from "../manifest/schema.js";

export interface RuntimeState {
  initialized: boolean;
  shuttingDown: boolean;
  protocolVersion: number;
  wireProtocol: string | null;
  manifest: ForstNodeManifestV1 | null;
  index: ManifestIndex | null;
}

interface HostInitSnapshot {
  fingerprint: string;
  manifest: ForstNodeManifestV1;
  index: ManifestIndex;
  wireProtocol: string;
}

let hostInitSnapshot: HostInitSnapshot | null = null;

/** Test-only: clears process-wide host initialize cache. */
export function resetHostInitCacheForTest(): void {
  hostInitSnapshot = null;
}

function initializeFingerprint(params: InitializeParams): string {
  return JSON.stringify({
    protocolVersion: params.protocolVersion,
    boundaryRoot: params.boundaryRoot,
    manifest: params.manifest,
    filesExclude: params.filesExclude ?? null,
    supportedProtocols: params.supportedProtocols ?? null,
  });
}

function exportAllowlistKey(exp: { moduleId: string; name: string }): string {
  return `${exp.moduleId}\0${exp.name}`;
}

/** True when incoming adds exports while retaining every frozen export (same boundary). */
function manifestWidensAllowlist(
  frozen: ForstNodeManifestV1,
  incoming: ForstNodeManifestV1
): boolean {
  const frozenKeys = new Set(frozen.exports.map(exportAllowlistKey));
  const incomingKeys = new Set(incoming.exports.map(exportAllowlistKey));
  for (const key of frozenKeys) {
    if (!incomingKeys.has(key)) {
      return false;
    }
  }
  return incoming.exports.some((exp) => !frozenKeys.has(exportAllowlistKey(exp)));
}

function applyHostInitSnapshot(
  state: RuntimeState,
  snap: HostInitSnapshot
): InitializeResult {
  state.manifest = snap.manifest;
  state.index = snap.index;
  state.wireProtocol = snap.wireProtocol;
  state.initialized = true;
  return { ok: true as const, protocol: snap.wireProtocol };
}

export function createRuntimeState(): RuntimeState {
  return {
    initialized: false,
    shuttingDown: false,
    protocolVersion: PROTOCOL_VERSION,
    wireProtocol: null,
    manifest: null,
    index: null,
  };
}

function resolveServerWireProtocols(): string[] {
  return [WIRE_PROTOCOL_PROTO_V1];
}

function pickWireProtocol(
  clientPrefs: string[] | undefined,
  serverPrefs: string[]
): string {
  if (clientPrefs === undefined || clientPrefs.length === 0) {
    return serverPrefs[0] ?? "";
  }
  const clientSet = new Set(clientPrefs);
  for (const protocol of serverPrefs) {
    if (protocol === WIRE_PROTOCOL_PROTO_V1 && clientSet.has(WIRE_PROTOCOL_PROTO_V1)) {
      return WIRE_PROTOCOL_PROTO_V1;
    }
  }
  return "";
}

export const initializeRuntime = Effect.fn("Runtime.initialize")(
  function* (state: RuntimeState, params: InitializeParams) {
    if (state.initialized) {
      return yield* Effect.fail(Errors.invalidParams("runtime already initialized"));
    }

    if (params.protocolVersion !== PROTOCOL_VERSION) {
      return yield* Effect.fail(
        Errors.invalidParams("unsupported protocol version", {
          expected: PROTOCOL_VERSION,
          received: params.protocolVersion,
        })
      );
    }

    const protocol = pickWireProtocol(
      params.supportedProtocols,
      resolveServerWireProtocols()
    );
    if (protocol === "") {
      return yield* Effect.fail(
        Errors.invalidParams("no mutually supported wire protocol", {
          supportedProtocols: params.supportedProtocols,
          serverProtocols: resolveServerWireProtocols(),
        })
      );
    }

    if (typeof params.boundaryRoot !== "string" || params.boundaryRoot === "") {
      return yield* Effect.fail(
        Errors.invalidParams("boundaryRoot must be a non-empty string")
      );
    }

    const fingerprint = initializeFingerprint(params);
    if (hostInitSnapshot?.fingerprint === fingerprint) {
      const result = applyHostInitSnapshot(state, hostInitSnapshot);
      yield* Effect.logInfo("initialize_cache_hit").pipe(
        Effect.annotateLogs({
          event: "initialize_cache_hit",
          boundary_root: params.boundaryRoot,
          export_count: hostInitSnapshot.manifest.exports.length,
          protocol: hostInitSnapshot.wireProtocol,
        })
      );
      return result;
    }

    const manifest = yield* Effect.try({
      try: () => validateManifest(params.manifest),
      catch: (cause) =>
        cause instanceof Errors.JsonRpcError ? cause : Errors.invalidParams(String(cause)),
    });
    if (manifest.boundaryRoot !== params.boundaryRoot) {
      return yield* Effect.fail(
        Errors.invalidParams("manifest.boundaryRoot must match boundaryRoot", {
          boundaryRoot: params.boundaryRoot,
          manifestBoundaryRoot: manifest.boundaryRoot,
        })
      );
    }

    if (
      hostInitSnapshot !== null &&
      hostInitSnapshot.manifest.boundaryRoot === manifest.boundaryRoot &&
      manifestWidensAllowlist(hostInitSnapshot.manifest, manifest)
    ) {
      return yield* Effect.fail(
        Errors.forbidden("initialize cannot widen export allowlist", {
          boundaryRoot: params.boundaryRoot,
        })
      );
    }

    state.manifest = manifest;
    state.index = buildManifestIndex(manifest);
    state.wireProtocol = protocol;
    setFilesExcludePatterns(params.filesExclude);
    state.initialized = true;

    hostInitSnapshot = {
      fingerprint,
      manifest,
      index: state.index,
      wireProtocol: protocol,
    };

    yield* Effect.annotateCurrentSpan("boundary_root", params.boundaryRoot);
    yield* Effect.annotateCurrentSpan("export_count", manifest.exports.length);
    yield* Effect.annotateCurrentSpan("protocol", protocol);

    yield* Effect.logInfo("initialize").pipe(
      Effect.annotateLogs({
        event: "initialize",
        boundary_root: params.boundaryRoot,
        export_count: manifest.exports.length,
        protocol,
      })
    );

    return { ok: true as const, protocol } satisfies InitializeResult;
  }
);

export function assertInitialized(state: RuntimeState): ManifestIndex {
  if (!state.initialized || state.index === null) {
    throw Errors.notInitialized();
  }
  return state.index;
}

export const shutdownRuntime = Effect.fn("Runtime.shutdown")(
  function* (state: RuntimeState) {
    state.shuttingDown = true;
    yield* Effect.logInfo("shutdown").pipe(
      Effect.annotateLogs({ event: "shutdown" })
    );
    return { ok: true as const };
  }
);

export function shouldExitAfterShutdown(state: RuntimeState): boolean {
  return state.shuttingDown;
}
