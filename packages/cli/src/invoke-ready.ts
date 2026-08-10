import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";

/**
 * JSON written to `.forst/invoke.ready` when an embedded/dev invoke server binds.
 * Lets clients discover the HTTP base URL (and optional Unix socket) without
 * hard-coding localhost. Does not contain the auth token.
 */
export interface InvokeReadyPayload {
  /** HTTP base URL when transport is TCP (trailing slash may be present). */
  url?: string;
  /** Absolute Unix socket path when transport is `unix`. */
  socketPath?: string;
  /** Live auth generation; required for proof headers. */
  generation?: number;
  /** Invoke HTTP contract revision string. */
  contractVersion?: string;
  /** Server runtime label (`embedded`, `dev`, …). */
  runtime?: string;
}

/**
 * Injectable filesystem surface for {@link readInvokeReadyUrl} and related readers.
 * Tests pass stubs; production uses `node:fs`.
 */
export interface ReadInvokeReadyUrlFs {
  existsSync: typeof existsSync;
  readFileSync: typeof readFileSync;
}

/**
 * Parses `.forst/invoke.ready` under `boundaryRoot`, or returns `undefined`
 * when the file is missing or invalid JSON.
 */
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

/**
 * Reads `boundaryRoot/.forst/invoke.ready` for the invoke HTTP base URL.
 *
 * @param boundaryRoot Project root containing `.forst/` (defaults to `process.cwd()`).
 * @param fs Optional filesystem overrides for tests.
 * @returns Stripped base URL, or `undefined` when missing or unreadable.
 */
export function readInvokeReadyUrl(
  boundaryRoot?: string,
  fs: ReadInvokeReadyUrlFs = { existsSync, readFileSync }
): string | undefined {
  const payload = readInvokeReadyPayload(boundaryRoot, fs);
  const url = payload?.url?.trim();
  return url ? url.replace(/\/$/, "") : undefined;
}

/**
 * Reads the optional Unix socket path from `invoke.ready`.
 *
 * @param boundaryRoot Project root containing `.forst/`.
 * @param fs Optional filesystem overrides for tests.
 * @returns Absolute socket path, or `undefined` when unset or missing.
 */
export function readInvokeReadySocketPath(
  boundaryRoot?: string,
  fs: ReadInvokeReadyUrlFs = { existsSync, readFileSync }
): string | undefined {
  const payload = readInvokeReadyPayload(boundaryRoot, fs);
  const socketPath = payload?.socketPath?.trim();
  return socketPath || undefined;
}

/**
 * Reads the auth generation counter from `invoke.ready`.
 *
 * @param boundaryRoot Project root containing `.forst/`.
 * @param fs Optional filesystem overrides for tests.
 * @returns Generation number, or `undefined` when unset or missing.
 */
export function readInvokeReadyGeneration(
  boundaryRoot?: string,
  fs: ReadInvokeReadyUrlFs = { existsSync, readFileSync }
): number | undefined {
  return readInvokeReadyPayload(boundaryRoot, fs)?.generation;
}

/**
 * Reads and base64url-decodes `.forst/invoke.token` under `boundaryRoot`.
 *
 * The token file is written when auth is enabled and memory handoff is not used.
 * Returns `undefined` when the file is missing or undecodable.
 *
 * @param boundaryRoot Project root containing `.forst/`.
 * @param fs Optional filesystem overrides for tests.
 * @returns Raw token bytes for HMAC proofs.
 */
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

/**
 * Loads token + generation (and optional URL / socket) for authenticated invoke.
 *
 * Requires both `invoke.ready` (with `generation`) and `invoke.token`.
 * Use with `fetchAuthenticatedInvoke` after the server is healthy.
 *
 * @param boundaryRoot Project root containing `.forst/`.
 * @param fs Optional filesystem overrides for tests.
 * @returns Auth bundle, or `undefined` when ready/token/generation are incomplete.
 */
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
