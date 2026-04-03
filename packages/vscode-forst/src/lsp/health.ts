import type { LogOutputChannel } from "vscode";
import { LspHealthTimeoutError } from "./errors";

/**
 * Polls `/health` so activation can distinguish “server not up yet” from hard JSON-RPC failures
 * when auto-start races or when connecting to an externally managed `forst lsp`.
 */
export async function waitForLspHealth(
  baseUrl: string,
  timeoutMs: number,
  log?: LogOutputChannel
): Promise<void> {
  const healthUrl = `${baseUrl.replace(/\/$/, "")}/health`;
  const started = Date.now();
  const deadline = started + timeoutMs;
  let lastErr: string | undefined;
  while (Date.now() < deadline) {
    try {
      const r = await fetch(healthUrl);
      if (r.ok) {
        log?.debug(
          `[health] ready in ${Date.now() - started}ms (${healthUrl})`
        );
        return;
      }
      lastErr = `HTTP ${r.status}`;
    } catch (e) {
      lastErr = e instanceof Error ? e.message : String(e);
    }
    await new Promise((r) => setTimeout(r, 100));
  }
  log?.error(
    `[health] timeout after ${timeoutMs}ms (${healthUrl}): ${lastErr ?? "unknown"}`
  );
  throw new LspHealthTimeoutError(healthUrl, lastErr);
}
