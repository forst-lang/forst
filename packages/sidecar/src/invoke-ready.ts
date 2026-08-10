import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";
import type { InvokeReadyAuthPayload } from "./invoke-auth";

function isNodeRuntime(): boolean {
  return (
    typeof process !== "undefined" &&
    typeof process.versions?.node === "string"
  );
}

function readInvokeReadyPayload(boundaryRoot?: string): InvokeReadyAuthPayload | undefined {
  if (!isNodeRuntime()) {
    return undefined;
  }
  const root = boundaryRoot ?? process.cwd();
  const readyPath = join(root, ".forst", "invoke.ready");
  if (!existsSync(readyPath)) {
    return undefined;
  }
  try {
    return JSON.parse(readFileSync(readyPath, "utf8")) as InvokeReadyAuthPayload;
  } catch {
    return undefined;
  }
}

/** Reads boundaryRoot/.forst/invoke.ready for the embedded invoke base URL (Node only). */
export interface InvokeReadyPayload {
  url?: string;
  socketPath?: string;
  generation?: number;
  contractVersion?: string;
  runtime?: string;
}

export function readInvokeReadyUrl(boundaryRoot?: string): string | undefined {
  const payload = readInvokeReadyPayload(boundaryRoot);
  const url = payload?.url?.trim();
  return url ? url.replace(/\/$/, "") : undefined;
}

export function readInvokeReadySocketPath(boundaryRoot?: string): string | undefined {
  const payload = readInvokeReadyPayload(boundaryRoot);
  const socketPath = payload?.socketPath?.trim();
  return socketPath || undefined;
}

export function readInvokeReadyGeneration(boundaryRoot?: string): number | undefined {
  return readInvokeReadyPayload(boundaryRoot)?.generation;
}

export function readInvokeTokenFile(boundaryRoot?: string): Uint8Array | undefined {
  if (!isNodeRuntime()) {
    return undefined;
  }
  const root = boundaryRoot ?? process.cwd();
  const tokenPath = join(root, ".forst", "invoke.token");
  if (!existsSync(tokenPath)) {
    return undefined;
  }
  try {
    const raw = readFileSync(tokenPath, "utf8").trim();
    return Uint8Array.from(Buffer.from(raw, "base64url"));
  } catch {
    return undefined;
  }
}

export function readInvokeReadyAuth(boundaryRoot?: string):
  | { token: Uint8Array; generation: number; url?: string; socketPath?: string }
  | undefined {
  const payload = readInvokeReadyPayload(boundaryRoot);
  const token = readInvokeTokenFile(boundaryRoot);
  if (!payload || !token || payload.generation === undefined) {
    return undefined;
  }
  return {
    token,
    generation: payload.generation,
    url: readInvokeReadyUrl(boundaryRoot),
    socketPath: readInvokeReadySocketPath(boundaryRoot),
  };
}
