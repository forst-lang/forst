import * as vscode from "vscode";
import type { LogOutputChannel } from "vscode";
import type { ForstExtensionConfig } from "../config";
import { lspBaseUrl } from "../config";
import { ForstHttpLspClient } from "./client";
import type { ForstLspChildState } from "./process";
import { ensureForstLspProcess } from "./process";

/**
 * Holds the HTTP client, initialize flag, and child-process handle together so restart can reset
 * all three consistently without scattering mutable module globals across files.
 */
export interface LspSessionState {
  client: ForstHttpLspClient | undefined;
  initialized: boolean;
  child: ForstLspChildState;
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
  await ensureForstLspProcess(cfg, log, session.child);
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
  }
  return session.client;
}
