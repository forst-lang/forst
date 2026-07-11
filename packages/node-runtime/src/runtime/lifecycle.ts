import {
  buildManifestIndex,
  validateManifest,
  type ManifestIndex,
} from "../policy/manifest.js";
import { setFilesExcludePatterns } from "../policy/paths.js";
import { invalidParams, notInitialized } from "../rpc/errors.js";
import {
  PROTOCOL_VERSION,
  WIRE_PROTOCOL_PROTO_V1,
  type InitializeParams,
  type InitializeResult,
} from "../rpc/protocol.js";
import type { ForstNodeManifestV1 } from "../manifest/schema.js";
import { log } from "../logging/logger.js";

export interface RuntimeState {
  initialized: boolean;
  shuttingDown: boolean;
  protocolVersion: number;
  wireProtocol: string | null;
  manifest: ForstNodeManifestV1 | null;
  index: ManifestIndex | null;
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

export function initializeRuntime(
  state: RuntimeState,
  params: InitializeParams
): InitializeResult {
  if (state.initialized) {
    throw invalidParams("runtime already initialized");
  }

  if (params.protocolVersion !== PROTOCOL_VERSION) {
    throw invalidParams("unsupported protocol version", {
      expected: PROTOCOL_VERSION,
      received: params.protocolVersion,
    });
  }

  const protocol = pickWireProtocol(
    params.supportedProtocols,
    resolveServerWireProtocols()
  );
  if (protocol === "") {
    throw invalidParams("no mutually supported wire protocol", {
      supportedProtocols: params.supportedProtocols,
      serverProtocols: resolveServerWireProtocols(),
    });
  }

  if (typeof params.boundaryRoot !== "string" || params.boundaryRoot === "") {
    throw invalidParams("boundaryRoot must be a non-empty string");
  }

  const manifest = validateManifest(params.manifest);
  if (manifest.boundaryRoot !== params.boundaryRoot) {
    throw invalidParams("manifest.boundaryRoot must match boundaryRoot", {
      boundaryRoot: params.boundaryRoot,
      manifestBoundaryRoot: manifest.boundaryRoot,
    });
  }

  state.manifest = manifest;
  state.index = buildManifestIndex(manifest);
  state.wireProtocol = protocol;
  setFilesExcludePatterns(params.filesExclude);
  state.initialized = true;

  log("initialize", {
    boundary_root: params.boundaryRoot,
    export_count: manifest.exports.length,
    protocol,
  });

  return { ok: true, protocol };
}

export function assertInitialized(state: RuntimeState): ManifestIndex {
  if (!state.initialized || state.index === null) {
    throw notInitialized();
  }
  return state.index;
}

export function shutdownRuntime(state: RuntimeState): { ok: true } {
  state.shuttingDown = true;
  log("shutdown", {});
  return { ok: true };
}

export function shouldExitAfterShutdown(state: RuntimeState): boolean {
  return state.shuttingDown;
}
