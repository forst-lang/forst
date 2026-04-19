import * as fs from "fs";
import * as path from "path";
import type { LogOutputChannel } from "vscode";
import * as vscode from "vscode";

/** Snapshot of `forst.*` workspace settings the LSP layer and spawn logic need in one place. */
export interface ForstExtensionConfig {
  forstPath: string;
  port: number;
  logLevel: string;
  autoStart: boolean;
  /**
   * When true (default), use @forst/cli to resolve FORST_BINARY, cache, or download the compiler.
   * When false, use workspace `bin/forst` / PATH only.
   */
  downloadCompiler: boolean;
  /**
   * When true (default) and downloads are enabled, compare the bundled @forst/cli semver with
   * GitHub's latest release and use the higher version for the cache path (may fetch a newer binary).
   */
  preferLatestCompilerRelease: boolean;
}

/** Reads the current workspace configuration—call sites get fresh values after settings change. */
export function readForstConfig(): ForstExtensionConfig {
  const cfg = vscode.workspace.getConfiguration("forst");
  const pathRaw = cfg.get<string>("compiler.path") ?? "";
  const port = cfg.get<number>("lsp.port") ?? 8081;
  const logLevel = cfg.get<string>("lsp.logLevel") ?? "info";
  const autoStart = cfg.get<boolean>("lsp.autoStart") ?? true;
  const downloadCompiler = cfg.get<boolean>("compiler.download") ?? true;
  const preferLatestCompilerRelease =
    cfg.get<boolean>("compiler.preferLatestRelease") ?? true;
  return {
    forstPath: pathRaw.trim() || "forst",
    port,
    logLevel,
    autoStart,
    downloadCompiler,
    preferLatestCompilerRelease,
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

const defaultExeName = () =>
  process.platform === "win32" ? "forst.exe" : "forst";

/**
 * Resolves the `forst` executable: explicit path, workspace `bin/forst`, then @forst/cli
 * (FORST_BINARY, cache, optional download per `downloadCompiler`).
 */
export async function resolveForstExecutableWithCli(
  cfg: ForstExtensionConfig,
  log?: LogOutputChannel
): Promise<string> {
  const def = defaultExeName();
  const resolved = resolveForstExecutable(cfg.forstPath);

  if (resolved !== def && resolved !== "forst") {
    if (!fs.existsSync(resolved)) {
      throw new Error(`forst executable not found: ${resolved}`);
    }
    return resolved;
  }

  try {
    const { resolveForstBinary } = await import("@forst/cli");
    const p = await resolveForstBinary({
      allowDownload: cfg.downloadCompiler,
      preferLatestRelease:
        cfg.downloadCompiler && cfg.preferLatestCompilerRelease,
    });
    if (p !== resolved) {
      log?.info(`Resolved Forst compiler via @forst/cli: ${p}`);
    }
    return p;
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    log?.warn(
      `Forst CLI could not resolve the compiler (${msg}). Using "${resolved}" (PATH).`
    );
    return resolved;
  }
}
