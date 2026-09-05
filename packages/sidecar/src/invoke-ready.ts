/**
 * Ready/token file readers. Implementation lives in `@forst/cli/invoke`.
 * Sidecar wraps with a Node-runtime guard so browser/edge bundles no-op safely.
 */
import {
  readInvokeReadyAuth as readInvokeReadyAuthImpl,
  readInvokeReadyGeneration as readInvokeReadyGenerationImpl,
  readInvokeReadySocketPath as readInvokeReadySocketPathImpl,
  readInvokeReadyUrl as readInvokeReadyUrlImpl,
  readInvokeTokenFile as readInvokeTokenFileImpl,
  type InvokeReadyPayload,
} from "@forst/cli/invoke";

export type { InvokeReadyPayload };

function isNodeRuntime(): boolean {
  return (
    typeof process !== "undefined" &&
    typeof process.versions?.node === "string"
  );
}

/** Reads boundaryRoot/.forst/invoke.ready for the embedded invoke base URL (Node only). */
export function readInvokeReadyUrl(boundaryRoot?: string): string | undefined {
  if (!isNodeRuntime()) {
    return undefined;
  }
  return readInvokeReadyUrlImpl(boundaryRoot);
}

export function readInvokeReadySocketPath(
  boundaryRoot?: string
): string | undefined {
  if (!isNodeRuntime()) {
    return undefined;
  }
  return readInvokeReadySocketPathImpl(boundaryRoot);
}

export function readInvokeReadyGeneration(
  boundaryRoot?: string
): number | undefined {
  if (!isNodeRuntime()) {
    return undefined;
  }
  return readInvokeReadyGenerationImpl(boundaryRoot);
}

export function readInvokeTokenFile(
  boundaryRoot?: string
): Uint8Array | undefined {
  if (!isNodeRuntime()) {
    return undefined;
  }
  return readInvokeTokenFileImpl(boundaryRoot);
}

export function readInvokeReadyAuth(boundaryRoot?: string):
  | { token: Uint8Array; generation: number; url?: string; socketPath?: string }
  | undefined {
  if (!isNodeRuntime()) {
    return undefined;
  }
  return readInvokeReadyAuthImpl(boundaryRoot);
}
