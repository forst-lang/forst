import { logger } from "./logger";

import type {
  ForstConfig,
  FunctionInfo,
  InvokeSuccess,
  ServerInfo,
  ServerVersionInfo,
  StreamingResult,
} from "./types";
import {
  CompilerNotFound,
  ConnectModeMissingUrl,
  ContractVersionMismatch,
  DevServerChildProcessNotResponding,
  GenerateCommandFailed,
  ServerVersionMismatch,
  SidecarNotStarted,
} from "./errors";
import { FORST_DEV_HTTP_CONTRACT_VERSION } from "./constants";
import {
  contractVersionCompatible,
  versionsEquivalentForSidecar,
} from "./version-compare";
import {
  mergeForstSidecarEnv,
  normalizeDevServerBaseUrl,
  parseDevServerUrlParts,
} from "./config-merge";
import { ForstUtils } from "./utils";
import { ForstSidecarClient } from "./client";
import {
  ForstServer,
  buildForstGenerateArgs,
  effectiveProjectRootDir,
} from "./server";

async function ensureForstBinary(config: ForstConfig): Promise<string> {
  return await ForstUtils.ensureCompiler(config.downloadCompiler ?? false);
}

/**
 * Main Forst sidecar class that provides the complete integration
 */
export class ForstSidecar {
  private server?: ForstServer;
  private client: ForstSidecarClient | null = null;
  private forstPath: string | null = null;
  private config: ForstConfig;
  /** When set, the sidecar is attached to an existing `forst dev` (no spawned child). */
  private connectBaseUrl: string | null = null;
  private _customCompilerPath: string | null = null; // Intentionally awkward - don't use this normally

  constructor(config?: Partial<ForstConfig>) {
    this.config = {
      mode: "development",
      port: 8080,
      host: "localhost",
      logLevel: "info",
      ...config,
    };
  }

  /**
   * Set a custom compiler path (INTENTIONALLY AWKWARD - don't use this normally)
   * This bypasses the normal binary resolution and should only be used for testing
   * @param path - The path to the custom Forst compiler binary
   */
  _setCustomCompilerPath(path: string): void {
    logger.warn(
      "⚠️  Using custom compiler path - this is intentionally awkward and not recommended for normal use"
    );
    this._customCompilerPath = path;
  }

  /**
   * Start the sidecar development server
   */
  async start(): Promise<void> {
    logger.info("🚀 Starting Forst sidecar...");
    this.config = mergeForstSidecarEnv(this.config);

    if (this.config.sidecarRuntime === "connect") {
      const raw = this.config.devServerUrl;
      if (!raw || raw.trim() === "") {
        throw new ConnectModeMissingUrl();
      }
      const baseUrl = normalizeDevServerBaseUrl(raw);
      this.connectBaseUrl = baseUrl;
      this.client = new ForstSidecarClient({
        baseUrl,
        timeout: 30000,
        retries: 1,
      });
      const ok = await this.client.healthCheck();
      if (!ok) {
        this.client = null;
        this.connectBaseUrl = null;
        throw new DevServerChildProcessNotResponding(
          `Could not reach existing Forst dev server at ${baseUrl}`
        );
      }
      logger.info(`✅ Forst sidecar connected to existing dev server (${baseUrl})`);
      await this.maybeCheckVersion();
      return;
    }

    this.connectBaseUrl = null;

    if (this._customCompilerPath) {
      this.forstPath = this._customCompilerPath;
      if (!this.forstPath) {
        throw new CompilerNotFound("Custom compiler path was empty.");
      }
      logger.info(`🔧 Using custom compiler path: ${this.forstPath}`);
    } else {
      try {
        this.forstPath = await ensureForstBinary(this.config);
      } catch (e) {
        throw new CompilerNotFound(
          "Failed to resolve the Forst compiler. Install `forst`, set FORST_BINARY, enable downloadCompiler (or FORST_DOWNLOAD_COMPILER=1), or use a cached @forst/cli binary.",
          { cause: e }
        );
      }
    }

    // Initialize server with the resolved forst path
    this.server = new ForstServer(this.config, this.forstPath);

    // Start the development server
    await this.server.start();

    // Initialize the client
    this.client = new ForstSidecarClient({
      baseUrl: this.server.getServerUrl(),
      timeout: 30000,
      retries: 1,
    });

    await this.maybeCheckVersion();
    logger.info("✅ Forst sidecar started successfully");
  }

  /**
   * Compare local `forst version` to `GET /version` when {@link ForstConfig.versionCheck} is not `off`.
   */
  private async maybeCheckVersion(): Promise<void> {
    const mode = this.config.versionCheck ?? "warn";
    if (mode === "off" || !this.client) {
      return;
    }
    let forstPath: string;
    try {
      forstPath =
        this._customCompilerPath ??
        this.forstPath ??
        (await ensureForstBinary(mergeForstSidecarEnv(this.config)));
    } catch {
      logger.warn(
        "Could not resolve local forst binary; skipping version check"
      );
      return;
    }
    const local = await ForstUtils.getLocalBinaryVersion(forstPath);
    if (!local) {
      logger.warn(
        "Could not read local forst version (`forst version`); skipping version check"
      );
      return;
    }
    let remote: ServerVersionInfo;
    try {
      remote = await this.client.getVersion();
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      if (mode === "strict") {
        throw e;
      }
      logger.warn(
        `Could not fetch dev server GET /version (older forst dev, or network): ${msg}`
      );
      return;
    }
    if (
      !contractVersionCompatible(
        remote.contractVersion,
        FORST_DEV_HTTP_CONTRACT_VERSION
      )
    ) {
      const detail = `Dev server HTTP contract version is ${remote.contractVersion}, this @forst/sidecar expects ${FORST_DEV_HTTP_CONTRACT_VERSION}`;
      if (mode === "strict") {
        throw new ContractVersionMismatch(
          detail,
          FORST_DEV_HTTP_CONTRACT_VERSION,
          remote.contractVersion
        );
      }
      logger.warn(detail);
    } else {
      logger.debug(
        `HTTP contract version OK (${remote.contractVersion})`
      );
    }
    if (!versionsEquivalentForSidecar(local, remote.version)) {
      const detail = `Local forst binary version is ${local}, dev server reports ${remote.version}`;
      if (mode === "strict") {
        throw new ServerVersionMismatch(detail, local, remote.version);
      }
      logger.warn(detail);
    } else {
      logger.debug(`Forst compiler version check OK: ${local}`);
    }
  }

  /**
   * Run `forst generate` on {@link ForstConfig.rootDir} / {@link ForstConfig.forstDir} (same tree as `forst dev -root`),
   * writing `generated/` TypeScript under that directory. Does not require the HTTP dev server.
   */
  async generateTypes(): Promise<void> {
    const cfg = mergeForstSidecarEnv(this.config);
    const root = effectiveProjectRootDir(cfg);
    const forstPath =
      this._customCompilerPath ??
      this.forstPath ??
      (await ensureForstBinary(cfg));
    const args = buildForstGenerateArgs(cfg, root);
    const { exitCode, stderr, stdout } = await ForstUtils.executeForstCommand(
      forstPath,
      args,
      { cwd: root }
    );
    if (exitCode !== 0) {
      throw new GenerateCommandFailed(
        `forst generate exited with code ${exitCode}`,
        stderr || stdout
      );
    }
    logger.info(`forst generate completed for ${root}`);
  }

  /**
   * Read compiler and contract metadata from the running dev server (`GET /version`).
   */
  async getVersion(): Promise<ServerVersionInfo> {
    if (!this.client) {
      throw new SidecarNotStarted();
    }
    return this.client.getVersion();
  }

  /**
   * Stop the sidecar development server
   */
  async stop(): Promise<void> {
    if (this.connectBaseUrl) {
      logger.info("🛑 Disconnecting Forst sidecar (connect mode)...");
      this.client = null;
      this.connectBaseUrl = null;
      logger.info("✅ Forst sidecar disconnected");
      return;
    }
    if (!this.server) {
      logger.debug("ForstSidecar.stop(): server was never started; skipping.");
      return;
    }
    logger.info("🛑 Stopping Forst sidecar...");
    await this.server.stop();
    this.client = null;
    logger.info("✅ Forst sidecar stopped");
  }

  /**
   * Get the client for making function calls
   */
  getClient(): ForstSidecarClient {
    if (!this.client) {
      throw new SidecarNotStarted();
    }
    return this.client;
  }

  /**
   * Get server information. Requires {@link start} to have completed successfully.
   */
  getServerInfo(): ServerInfo {
    if (this.connectBaseUrl) {
      const { host, port } = parseDevServerUrlParts(this.connectBaseUrl);
      return {
        pid: 0,
        port,
        host,
        status: "running",
        connection: "connect",
      };
    }
    if (!this.server) {
      throw new SidecarNotStarted();
    }
    return this.server.getServerInfo();
  }

  /**
   * Whether the underlying dev server process is running. Returns false if {@link start} has not run.
   */
  isRunning(): boolean {
    if (this.connectBaseUrl) {
      return this.client !== null;
    }
    if (!this.server) {
      return false;
    }
    return this.server.isRunning();
  }

  /**
   * Discover available functions
   */
  async discoverFunctions(): Promise<FunctionInfo[]> {
    if (!this.client) {
      throw new SidecarNotStarted();
    }
    return this.client.discoverFunctions();
  }

  /**
   * Invoke a Forst function with **positional** arguments. The dev server passes `args` as JSON
   * to the executor (same shape as `POST /invoke`: typically a JSON array matching Forst parameter order).
   */
  async invoke(
    packageName: string,
    functionName: string,
    args: unknown[] = []
  ): Promise<InvokeSuccess<unknown>> {
    if (!this.client) {
      throw new SidecarNotStarted();
    }
    return this.client.invokeFunction(packageName, functionName, args as any[]);
  }

  /**
   * Streaming invoke; same **`args`** as {@link invoke}. See {@link ForstSidecarClient.invokeStream}.
   */
  invokeStream<T = unknown>(
    packageName: string,
    functionName: string,
    args?: unknown[]
  ): AsyncGenerator<StreamingResult & { data?: T }, void, undefined>;
  invokeStream(
    packageName: string,
    functionName: string,
    args: unknown[] | undefined,
    onResult: (result: StreamingResult) => void | Promise<void>
  ): Promise<void>;
  invokeStream<T = unknown>(
    packageName: string,
    functionName: string,
    args: unknown[] = [],
    onResult?: (result: StreamingResult) => void | Promise<void>
  ): Promise<void> | AsyncGenerator<StreamingResult & { data?: T }, void, undefined> {
    if (!this.client) {
      throw new SidecarNotStarted();
    }
    if (onResult !== undefined) {
      return this.client.invokeStream(packageName, functionName, args, onResult);
    }
    return this.client.invokeStream<T>(packageName, functionName, args);
  }

  /**
   * Health check
   */
  async healthCheck(): Promise<boolean> {
    if (!this.client) {
      return false;
    }
    return this.client.healthCheck();
  }
}

export default ForstSidecar;
