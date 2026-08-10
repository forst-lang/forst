import { existsSync } from "node:fs";
import { resolve } from "node:path";
import chokidar from "chokidar";
import { ForstConfig, ServerInfo } from "./types";
import { ForstUtils } from "./utils";
import { serverLogger, forstLogger } from "./logger";
import { ProcessSupervisor } from "./process-supervisor";

/**
 * Resolves the project root directory for `forst dev -root` and the supervised child cwd.
 *
 * @param cfg Sidecar configuration; prefers `rootDir`, then `forstDir`, else `./forst`.
 * @returns Absolute project root path.
 */
export function effectiveProjectRootDir(cfg: ForstConfig): string {
  return resolve(cfg.rootDir ?? cfg.forstDir ?? "./forst");
}

/**
 * Default directory used for `.ft` watch when {@link ForstConfig.watchRoots} is not set.
 * Prefers `forstDir`, then `rootDir`, then {@link effectiveProjectRootDir}.
 *
 * @param cfg Sidecar configuration whose directory fields define the watch root.
 * @returns Absolute path to the primary watch directory.
 */
export function effectiveWatchDirForConfig(cfg: ForstConfig): string {
  return resolve(
    cfg.forstDir ?? cfg.rootDir ?? effectiveProjectRootDir(cfg)
  );
}

/**
 * Absolute watch roots: either explicit `watchRoots` or a single {@link effectiveWatchDirForConfig}.
 * Skips paths that do not exist on disk.
 *
 * @param cfg Sidecar configuration; uses {@link ForstConfig.watchRoots} when set.
 * @returns Existing directories to pass to the file watcher (may be empty).
 */
export function buildForstWatchRoots(cfg: ForstConfig): string[] {
  if (cfg.watchRoots && cfg.watchRoots.length > 0) {
    return cfg.watchRoots.map((r) => resolve(r)).filter((p) => existsSync(p));
  }
  const dir = effectiveWatchDirForConfig(cfg);
  if (!existsSync(dir)) {
    return [];
  }
  return [dir];
}

/**
 * Arguments and cwd for `spawn(forst, args, { cwd })` to run `forst dev`.
 * Exposed for unit tests and advanced integrations.
 *
 * @param cfg Sidecar configuration (root, log level, optional `ftconfig.json`).
 * @param port TCP port passed to `forst dev -port`.
 * @returns argv for the dev subcommand and cwd for the child process.
 */
export function buildForstDevSpawnArgs(
  cfg: ForstConfig,
  port: number
): { args: string[]; cwd: string } {
  const cwd = effectiveProjectRootDir(cfg);
  const args: string[] = [
    "dev",
    "-port",
    String(port),
    "-root",
    cwd,
    "-log-level",
    cfg.logLevel || "info",
  ];
  if (cfg.configPath) {
    args.push("-config", resolve(cfg.configPath));
  }
  return { args, cwd };
}

/**
 * Arguments for `forst generate`, aligned with {@link buildForstDevSpawnArgs}: optional `-config`, then project root.
 *
 * @param cfg Sidecar configuration supplying an optional config file path.
 * @param root Project root directory passed as the final generate argument.
 * @returns argv for the generate subcommand.
 */
export function buildForstGenerateArgs(
  cfg: ForstConfig,
  root: string
): string[] {
  const args: string[] = ["generate"];
  if (cfg.configPath) {
    args.push("-config", resolve(cfg.configPath));
  }
  args.push(root);
  return args;
}

/**
 * Spawns and supervises `forst dev`, exposes the listen URL via {@link ForstServer.getServerUrl},
 * watches `.ft` files for reload, and captures invoke auth from the child when available.
 */
export class ForstServer {
  private supervisor: ProcessSupervisor;
  private config: ForstConfig;
  private forstPath: string;
  private status: ServerInfo["status"] = "stopped";
  private port: number;
  private host: string;
  private fileWatchers: Array<() => Promise<void>> = [];
  private shutdownHandler: () => void;

  /**
   * @param config Sidecar configuration (port, host, watch roots, `ftconfig.json`, etc.).
   * @param forstPath Absolute path to the `forst` binary used for `forst dev` and `forst generate`.
   */
  constructor(config: ForstConfig, forstPath: string) {
    this.config = config;
    this.forstPath = forstPath;
    this.port = config.port || 6320;
    this.host = config.host || "localhost";
    this.supervisor = new ProcessSupervisor(forstPath, this.host, this.port);

    // Set up interrupt handlers (sync wrapper so Node never leaves a rejected Promise from an async listener).
    this.shutdownHandler = (): void => {
      void (async () => {
        serverLogger.info(
          "Received interrupt signal, shutting down gracefully..."
        );
        try {
          await this.stop();
          process.exit(0);
        } catch (error) {
          serverLogger.error("Error during shutdown:", error);
          process.exit(1);
        }
      })().catch((err) => {
        serverLogger.error("Fatal shutdown error:", err);
        process.exit(1);
      });
    };
  }

  /**
   * Spawns the supervised `forst dev` child, waits until it responds to health checks,
   * and starts `.ft` file watching when watch roots exist.
   *
   * Registers SIGINT/SIGTERM handlers that call {@link stop} before exit.
   * Idempotent while already running (returns current {@link ServerInfo}).
   *
   * @returns Snapshot of pid, listen address, and lifecycle status after startup.
   * @throws When the child fails to start or never becomes healthy within the supervisor timeout.
   */
  async start(): Promise<ServerInfo> {
    if (this.status === "running") {
      return this.getServerInfo();
    }

    this.status = "starting";

    process.on("SIGINT", this.shutdownHandler);
    process.on("SIGTERM", this.shutdownHandler);

    try {
      // Start the server process using the resolved forstPath
      await this.startServerProcess();

      // Set up file watching
      await this.setupFileWatching();

      this.status = "running";
      forstLogger.info(
        `🚀 Forst development server started on http://${this.host}:${this.port}`
      );

      return this.getServerInfo();
    } catch (error) {
      this.status = "error";
      serverLogger.error("Failed to start Forst server:", error);
      throw error;
    }
  }

  /**
   * Stops file watchers, sends SIGTERM to the supervised child (SIGKILL on timeout),
   * and removes signal handlers registered by {@link start}.
   *
   * No-op when already stopped.
   */
  async stop(): Promise<void> {
    if (this.status === "stopped") {
      return;
    }

    this.status = "stopped";

    // Remove interrupt handlers
    process.off("SIGINT", this.shutdownHandler);
    process.off("SIGTERM", this.shutdownHandler);

    // Stop file watchers
    for (const close of this.fileWatchers) {
      try {
        await close();
      } catch (e) {
        serverLogger.warn("Error closing file watcher:", e);
      }
    }
    this.fileWatchers = [];

    // Stop supervised child process
    await this.supervisor.stop();

    forstLogger.info("🛑 Forst development server stopped");
  }

  /**
   * {@link stop} followed by {@link start}. Used after debounced `.ft` file changes.
   *
   * @returns Fresh {@link ServerInfo} from the new child process.
   */
  async restart(): Promise<ServerInfo> {
    serverLogger.info("🔄 Restarting Forst development server...");
    await this.stop();
    return this.start();
  }

  /**
   * Current embedded dev-server snapshot for logging and sidecar status APIs.
   *
   * @returns pid of the supervised child (0 if not running), configured host/port,
   *   coarse lifecycle status, and `connection: "spawn"`.
   */
  getServerInfo(): ServerInfo {
    return {
      pid: this.supervisor.child?.pid || 0,
      port: this.port,
      host: this.host,
      status: this.status,
      connection: "spawn",
    };
  }

  /**
   * Start the server process
   */
  private async startServerProcess(): Promise<void> {
    const port = this.config.port || 6320;
    const plan = buildForstDevSpawnArgs(this.config, port);
    if (this.config.configPath) {
      serverLogger.debug(
        `Using explicit Forst config file: ${resolve(this.config.configPath)}`
      );
    }
    await this.supervisor.start(plan);
    this.status = this.supervisor.processStatus;
  }

  /**
   * Set up file watching for hot reloading (chokidar; ignores node_modules / .git).
   */
  private async setupFileWatching(): Promise<void> {
    const roots = buildForstWatchRoots(this.config);
    if (roots.length === 0) {
      serverLogger.debug(
        "No watch roots (directories missing or empty); skipping file watch."
      );
      return;
    }

    serverLogger.debug(
      `Watching Forst sources under: ${roots.join(", ")}`
    );

    const watcher = chokidar.watch(roots, {
      ignored: [
        "**/node_modules/**",
        "**/.git/**",
        "**/dist/**",
      ],
      ignoreInitial: true,
      persistent: true,
    });

    watcher.on("error", (err) => {
      serverLogger.warn("File watcher error:", err);
    });

    watcher.on("all", (_event, filePath) => {
      if (filePath && filePath.endsWith(".ft")) {
        forstLogger.info(
          `📝 Detected change in ${filePath}, triggering reload...`
        );
        this.handleFileChange();
      }
    });

    this.fileWatchers.push(async () => {
      await watcher.close();
    });
  }

  /**
   * Handle file changes
   */
  private handleFileChange(): void {
    // Debounce file changes
    if (this.fileChangeTimeout) {
      clearTimeout(this.fileChangeTimeout);
    }
    this.fileChangeTimeout = setTimeout(() => {
      this.restart()
        .then(() => this.maybeGenerateAfterWatch())
        .catch((error) => {
          serverLogger.error(
            "Failed to restart server after file change:",
            error
          );
        });
    }, 1000);
  }

  /** When {@link ForstConfig.watchGenerate} is set, run `forst generate` after a debounced restart. */
  private async maybeGenerateAfterWatch(): Promise<void> {
    if (!this.config.watchGenerate) {
      return;
    }
    const root = effectiveProjectRootDir(this.config);
    const args = buildForstGenerateArgs(this.config, root);
    serverLogger.debug(`watchGenerate: ${this.forstPath} ${args.join(" ")}`);
    const { exitCode, stderr, stdout } = await ForstUtils.executeForstCommand(
      this.forstPath,
      args,
      { cwd: root }
    );
    if (exitCode !== 0) {
      serverLogger.warn(
        `watchGenerate: forst generate failed (exit ${exitCode}): ${stderr || stdout}`
      );
    } else {
      forstLogger.info("watchGenerate: forst generate completed");
    }
  }

  private fileChangeTimeout: NodeJS.Timeout | null = null;

  /**
   * HTTP base URL of the embedded invoke server (`http://host:port`).
   *
   * Uses the configured {@link ForstConfig.host} and {@link ForstConfig.port}
   * (defaults `localhost:6320`). Does not read `.forst/invoke.ready`; callers
   * that need post-reload discovery should read that file instead.
   */
  getServerUrl(): string {
    return `http://${this.host}:${this.port}`;
  }

  /**
   * Invoke auth secret delivered by the child over `FORST_INVOKE_AUTH_FD` after spawn.
   *
   * Populated asynchronously once the child writes its handoff line during startup.
   * Returns `null` before handoff completes, when handoff failed, or after {@link stop}.
   *
   * Prefer this over reading `.forst/invoke.token` in spawn mode so the token never
   * hits disk. Pair with {@link getServerUrl} or ready-file metadata for RPC transport setup.
   *
   * @returns Live token bytes and generation for HMAC invoke proofs, or `null` when unavailable.
   */
  getAuthHandoff(): { token: Uint8Array; generation: number } | null {
    const handoff = this.supervisor.auth;
    return handoff
      ? { token: handoff.token, generation: handoff.generation }
      : null;
  }

  /**
   * Whether the sidecar considers the dev server running and the child handle is present.
   *
   * True only when internal status is `"running"` and the supervised process has not exited.
   */
  isRunning(): boolean {
    return this.status === "running" && this.supervisor.child !== null;
  }
}
