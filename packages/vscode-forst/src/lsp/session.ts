import * as vscode from "vscode";
import type { LogOutputChannel } from "vscode";
import type { ForstExtensionConfig } from "../config";
import { lspBaseUrl } from "../config";
import { ForstHttpLspClient } from "./client";
import type { ForstLspChildState } from "./process";
import { ensureForstLspProcess } from "./process";
import { refreshForstStatusBar } from "../statusBar";

/**
 * Holds the HTTP client, initialize flag, and child-process handle together so restart can reset
 * all three consistently without scattering mutable module globals across files.
 */
export interface LspSessionState {
  client: ForstHttpLspClient | undefined;
  initialized: boolean;
  child: ForstLspChildState;
  /** Set after the first compiler resolution or spawn failure; cleared on restart. */
  compilerUnavailable?: Error;
}

/**
 * Ensures the compiler is reachable, (re)constructs the client when the port/base changes, and
 * performs `initialize` once per logical session—shared by document listeners and commands.
 */
export async function getOrCreateLspClient(
  cfg: ForstExtensionConfig,
  log: LogOutputChannel,
  session: LspSessionState
): Promise<ForstHttpLspClient> {
  if (session.compilerUnavailable) {
    throw session.compilerUnavailable;
  }
  try {
    await ensureForstLspProcess(cfg, log, session.child, session);
  } catch (e) {
    const err = e instanceof Error ? e : new Error(String(e));
    session.compilerUnavailable = err;
    throw err;
  }
  const base = lspBaseUrl(cfg.port);
  if (!session.client || session.client.base !== base) {
    log.debug(`[lsp] new HTTP client for ${base}`);
    session.client = new ForstHttpLspClient(base, log);
    session.initialized = false;
  }
  if (!session.initialized) {
    const root =
      vscode.workspace.workspaceFolders?.[0]?.uri.toString() ?? null;
    await session.client.initialize(root);
    session.initialized = true;
    log.info("LSP initialize completed");
    refreshForstStatusBar("ready");
  }
  return session.client;
}

/** Clears LSP client state and compiler failure latch (e.g. on restart or settings change). */
export function resetLspSessionState(session: LspSessionState): void {
  session.initialized = false;
  session.client = undefined;
  session.compilerUnavailable = undefined;
}
