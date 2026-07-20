import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";

/**
 * JSON written to `.forst/invoke.ready` when the dev server binds its invoke port;
 * lets sidecar/client discover the HTTP base URL without hard-coding localhost.
 */
export interface InvokeReadyPayload {
  url?: string;
  contractVersion?: string;
  runtime?: string;
}

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
  const root = boundaryRoot ?? process.cwd();
  const readyPath = join(root, ".forst", "invoke.ready");
  if (!existsSync(readyPath)) {
    return undefined;
  }
  try {
    const raw = readFileSync(readyPath, "utf8");
    const payload = JSON.parse(raw) as InvokeReadyPayload;
    const url = payload.url?.trim();
    return url ? url.replace(/\/$/, "") : undefined;
  } catch {
    return undefined;
  }
}
