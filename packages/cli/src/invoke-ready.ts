import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";

/**
 * JSON written to `.forst/invoke.ready` when an embedded/dev invoke server binds;
 * lets clients discover the HTTP base URL without hard-coding localhost.
 */
export interface InvokeReadyPayload {
  url?: string;
  contractVersion?: string;
  runtime?: string;
}

export interface ReadInvokeReadyUrlFs {
  existsSync: typeof existsSync;
  readFileSync: typeof readFileSync;
}

/** Reads boundaryRoot/.forst/invoke.ready for the invoke HTTP base URL. */
export function readInvokeReadyUrl(
  boundaryRoot?: string,
  fs: ReadInvokeReadyUrlFs = { existsSync, readFileSync }
): string | undefined {
  const root = boundaryRoot ?? process.cwd();
  const readyPath = join(root, ".forst", "invoke.ready");
  if (!fs.existsSync(readyPath)) {
    return undefined;
  }
  try {
    const raw = fs.readFileSync(readyPath, "utf8");
    const payload = JSON.parse(raw) as InvokeReadyPayload;
    const url = payload.url?.trim();
    return url ? url.replace(/\/$/, "") : undefined;
  } catch {
    return undefined;
  }
}
