import * as cp from "child_process";
import type { LogOutputChannel } from "vscode";
import type { ForstExtensionConfig } from "../config";
import { lspBaseUrl, resolveForstExecutableWithCli } from "../config";
import { waitForLspHealth } from "./health";

/**
 * Shared mutable handle for the spawned `forst lsp` child so restart and deactivate can tear
 * down the same process without a global singleton outside the extension host lifecycle.
 */
export interface ForstLspChildState {
  process: cp.ChildProcess | undefined;
}

/** Resolves once `spawn` succeeds or rejects on immediate spawn failure (missing binary, etc.). */
function waitForChildSpawn(proc: cp.ChildProcess): Promise<void> {
  return new Promise((resolve, reject) => {
    proc.once("error", reject);
    proc.once("spawn", () => resolve());
  });
}

/** Stops the managed child, if any—idempotent for safe repeated calls from restart/deactivate. */
export function stopForstLspProcess(
  state: ForstLspChildState,
  log?: LogOutputChannel
): void {
  if (state.process) {
    try {
      state.process.kill();
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      log?.warn(`forst lsp: could not kill child process — ${msg}`);
    }
    state.process = undefined;
  }
}

/** Serializes startup so parallel `getOrCreateLspClient` calls cannot spawn multiple `forst lsp` on the same port. */
let ensureChain: Promise<void> = Promise.resolve();

/**
 * Ensures a listening HTTP LSP exists: either waits on an existing server or spawns `forst lsp`
 * when `autoStart` is enabled and no child is already tracked.
 */
export async function ensureForstLspProcess(
  cfg: ForstExtensionConfig,
  log: LogOutputChannel,
  state: ForstLspChildState
): Promise<void> {
  const prev = ensureChain;
  let release!: () => void;
  ensureChain = new Promise<void>((r) => {
    release = r;
  });
  await prev;
  try {
    await ensureForstLspProcessUnlocked(cfg, log, state);
  } finally {
    release();
  }
}

async function ensureForstLspProcessUnlocked(
  cfg: ForstExtensionConfig,
  log: LogOutputChannel,
  state: ForstLspChildState
): Promise<void> {
  const base = lspBaseUrl(cfg.port);
  if (!cfg.autoStart) {
    log.debug(`[lsp] waiting for existing server at ${base} (autoStart off)`);
    await waitForLspHealth(base, 8000, log);
    log.info(`Using existing Forst LSP at ${base}`);
    return;
  }
  if (state.process) {
    log.debug(`[lsp] child already running; checking health at ${base}`);
    await waitForLspHealth(base, 2000, log);
    return;
  }
  // Another window, a manual `forst lsp`, or a previous session may already listen on this port.
  try {
    await waitForLspHealth(base, 600, undefined);
    log.info(
      `Forst LSP already listening at ${base}; not spawning (reuse existing server or change **forst.lsp.port**).`
    );
    return;
  } catch {
    /* port free or server not up yet — spawn below */
  }
  const exe = await resolveForstExecutableWithCli(cfg, log);
  if (exe !== cfg.forstPath) {
    log.info(`Resolved forst executable: ${exe}`);
  }
  log.info(
    `Starting: ${exe} lsp -port ${cfg.port} -log-level ${cfg.logLevel}`
  );
  state.process = cp.spawn(
    exe,
    ["lsp", "-port", String(cfg.port), "-log-level", cfg.logLevel],
    {
      stdio: ["ignore", "pipe", "pipe"],
      env: process.env,
    }
  );
  state.process.stdout?.on("data", (b: Buffer) => {
    log.append(b.toString());
  });
  state.process.stderr?.on("data", (b: Buffer) => {
    log.append(b.toString());
  });
  state.process.on("error", (err: NodeJS.ErrnoException) => {
    log.error(`forst process error: ${err.message}`);
  });
  state.process.on("exit", (code) => {
    if (code === 0) {
      log.info("forst lsp exited with code 0");
    } else {
      log.warn(
        `forst lsp exited with code ${code ?? "null"}. If the log shows "address already in use", set **forst.lsp.port** to a free port, disable **forst.lsp.autoStart** and run \`forst lsp\` yourself, or stop the other process on that port.`
      );
    }
    state.process = undefined;
  });
  try {
    await waitForChildSpawn(state.process);
  } catch (err) {
    state.process = undefined;
    const msg = err instanceof Error ? err.message : String(err);
    log.error(
      `Could not spawn forst (${exe}): ${msg}. Set **forst.compiler.path**, put forst on PATH, enable **forst.compiler.download**, or open a workspace that contains **bin/forst** after \`task build\`.`
    );
    throw err;
  }
  await waitForLspHealth(base, 15000, log);
  log.info(`Forst LSP ready at ${base}`);
}
