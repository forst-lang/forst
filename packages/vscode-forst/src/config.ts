import * as fs from "fs";
import * as path from "path";
import * as vscode from "vscode";

/** Snapshot of `forst.*` workspace settings the LSP layer and spawn logic need in one place. */
export interface ForstExtensionConfig {
  forstPath: string;
  port: number;
  logLevel: string;
  autoStart: boolean;
}

/** Reads the current workspace configuration—call sites get fresh values after settings change. */
export function readForstConfig(): ForstExtensionConfig {
  const cfg = vscode.workspace.getConfiguration("forst");
  const pathRaw = cfg.get<string>("path") ?? "";
  const port = cfg.get<number>("lsp.port") ?? 8081;
  const logLevel = cfg.get<string>("lsp.logLevel") ?? "info";
  const autoStart = cfg.get<boolean>("lsp.autoStart") ?? true;
  return {
    forstPath: pathRaw.trim() || "forst",
    port,
    logLevel,
    autoStart,
  };
}

/** Builds the loopback URL the HTTP JSON-RPC client and health checks use for a given port. */
export function lspBaseUrl(port: number): string {
  return `http://127.0.0.1:${port}`;
}

/**
 * Lets contributors run against `bin/forst` from a clone without editing settings, while still
 * honoring explicit paths when provided.
 */
export function resolveForstExecutable(forstPath: string): string {
  const p = forstPath.trim();
  const defaultName = process.platform === "win32" ? "forst.exe" : "forst";
  if (p !== "forst" && p !== "forst.exe") {
    return p;
  }
  const folders = vscode.workspace.workspaceFolders;
  if (!folders?.length) {
    return p;
  }
  for (const folder of folders) {
    let dir = folder.uri.fsPath;
    for (let i = 0; i < 16; i++) {
      const candidate = path.join(dir, "bin", defaultName);
      try {
        if (fs.existsSync(candidate)) {
          return candidate;
        }
      } catch {
        /* ignore */
      }
      const parent = path.dirname(dir);
      if (parent === dir) {
        break;
      }
      dir = parent;
    }
  }
  return p;
}
