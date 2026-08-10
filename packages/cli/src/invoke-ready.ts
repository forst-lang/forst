import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";

/**
 * JSON written to `.forst/invoke.ready` when an embedded/dev invoke server binds;
 * lets clients discover the HTTP base URL without hard-coding localhost.
 */
export interface InvokeReadyPayload {
  url?: string;
  socketPath?: string;
  generation?: number;
  contractVersion?: string;
  runtime?: string;
}

export interface ReadInvokeReadyUrlFs {
  existsSync: typeof existsSync;
  readFileSync: typeof readFileSync;
}

function readInvokeReadyPayload(
  boundaryRoot: string | undefined,
  fs: ReadInvokeReadyUrlFs
): InvokeReadyPayload | undefined {
  const root = boundaryRoot ?? process.cwd();
  const readyPath = join(root, ".forst", "invoke.ready");
  if (!fs.existsSync(readyPath)) {
    return undefined;
  }
  try {
    return JSON.parse(fs.readFileSync(readyPath, "utf8")) as InvokeReadyPayload;
  } catch {
    return undefined;
  }
}

/** Reads boundaryRoot/.forst/invoke.ready for the invoke HTTP base URL. */
export function readInvokeReadyUrl(
  boundaryRoot?: string,
  fs: ReadInvokeReadyUrlFs = { existsSync, readFileSync }
): string | undefined {
  const payload = readInvokeReadyPayload(boundaryRoot, fs);
  const url = payload?.url?.trim();
  return url ? url.replace(/\/$/, "") : undefined;
}

export function readInvokeReadySocketPath(
  boundaryRoot?: string,
  fs: ReadInvokeReadyUrlFs = { existsSync, readFileSync }
): string | undefined {
  const payload = readInvokeReadyPayload(boundaryRoot, fs);
  const socketPath = payload?.socketPath?.trim();
  return socketPath || undefined;
}

export function readInvokeReadyGeneration(
  boundaryRoot?: string,
  fs: ReadInvokeReadyUrlFs = { existsSync, readFileSync }
): number | undefined {
  return readInvokeReadyPayload(boundaryRoot, fs)?.generation;
}

export function readInvokeTokenFile(
  boundaryRoot?: string,
  fs: ReadInvokeReadyUrlFs = { existsSync, readFileSync }
): Uint8Array | undefined {
  const root = boundaryRoot ?? process.cwd();
  const tokenPath = join(root, ".forst", "invoke.token");
  if (!fs.existsSync(tokenPath)) {
    return undefined;
  }
  try {
    const raw = fs.readFileSync(tokenPath, "utf8").trim();
    return Uint8Array.from(Buffer.from(raw, "base64url"));
  } catch {
    return undefined;
  }
}

export function readInvokeReadyAuth(
  boundaryRoot?: string,
  fs: ReadInvokeReadyUrlFs = { existsSync, readFileSync }
):
  | { token: Uint8Array; generation: number; url?: string; socketPath?: string }
  | undefined {
  const payload = readInvokeReadyPayload(boundaryRoot, fs);
  const token = readInvokeTokenFile(boundaryRoot, fs);
  if (!payload || !token || payload.generation === undefined) {
    return undefined;
  }
  return {
    token,
    generation: payload.generation,
    url: readInvokeReadyUrl(boundaryRoot, fs),
    socketPath: readInvokeReadySocketPath(boundaryRoot, fs),
  };
}
