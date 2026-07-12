import { existsSync } from "node:fs";
import { resolve } from "node:path";
import chokidar from "chokidar";
import { ForstConfig, ServerInfo } from "./types";
import { ForstUtils } from "./utils";
import { serverLogger, forstLogger } from "./logger";
import { ProcessSupervisor } from "./process-supervisor";

/** Project root passed to `forst dev -root` and used as the child process cwd. */
export function effectiveProjectRootDir(cfg: ForstConfig): string {
  return resolve(cfg.rootDir ?? cfg.forstDir ?? "./forst");
}

/**
 * Default directory used for `.ft` watch when {@link ForstConfig.watchRoots} is not set.
 * Prefers `forstDir`, then `rootDir`, then {@link effectiveProjectRootDir}.
 */
export function effectiveWatchDirForConfig(cfg: ForstConfig): string {
  return resolve(
    cfg.forstDir ?? cfg.rootDir ?? effectiveProjectRootDir(cfg)
  );
}

/**
 * Absolute watch roots: either explicit `watchRoots` or a single {@link effectiveWatchDirForConfig}.
 * Skips paths that do not exist on disk.
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
 * Spawns and supervises `forst dev`, exposes the listen URL via `getServerUrl()`, and watches `.ft` files for reload.
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
   * Start the Forst development server
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
   * Stop the Forst development server
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
   * Restart the server
   */
  async restart(): Promise<ServerInfo> {
    serverLogger.info("🔄 Restarting Forst development server...");
    await this.stop();
    return this.start();
  }

  /**
   * Get server information
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
   * Get the server URL
   */
  getServerUrl(): string {
    return `http://${this.host}:${this.port}`;
  }

  /**
   * Check if server is running
   */
  isRunning(): boolean {
    return this.status === "running" && this.supervisor.child !== null;
  }
}
