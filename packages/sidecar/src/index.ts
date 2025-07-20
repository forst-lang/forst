import { logger } from "./logger";

export { ForstSidecarClient } from "./client";
export { ForstServer } from "./server";
export { ForstUtils } from "./utils";
export * from "./types";

import type { ForstConfig } from "./types";
import { ForstUtils } from "./utils";
import { ForstSidecarClient } from "./client";
import { ForstServer } from "./server";

// Function to ensure Forst binary is available
async function ensureForstBinary(): Promise<string> {
  return await ForstUtils.ensureCompiler();
}

/**
 * Main Forst sidecar class that provides the complete integration
 */
export class ForstSidecar {
  private server!: ForstServer;
  private client: ForstSidecarClient | null = null;
  private forstPath: string | null = null;
  private config: ForstConfig;
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

    // Use custom path if set (awkward way)
    if (this._customCompilerPath) {
      this.forstPath = this._customCompilerPath;
      logger.info(`🔧 Using custom compiler path: ${this.forstPath}`);
    } else {
      // Ensure Forst binary is available (normal way)
      this.forstPath = await ensureForstBinary();
    }

    // Check if Forst compiler is available
    if (!this.forstPath) {
      throw new Error(
        "Forst compiler not found. Please ensure the Forst compiler is installed and available in your PATH."
      );
    }

    // Initialize server with the resolved forst path
    this.server = new ForstServer(this.config, this.forstPath);

    // Start the development server
    await this.server.start();

    // Initialize the client
    this.client = new ForstSidecarClient({
      baseUrl: this.server.getServerUrl(),
      timeout: 30000,
      retries: 3,
    });

    logger.info("✅ Forst sidecar started successfully");
  }

  /**
   * Stop the sidecar development server
   */
  async stop(): Promise<void> {
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
      throw new Error("Sidecar not started. Call start() first.");
    }
    return this.client;
  }

  /**
   * Get server information
   */
  getServerInfo() {
    return this.server.getServerInfo();
  }

  /**
   * Check if the sidecar is running
   */
  isRunning(): boolean {
    return this.server.isRunning();
  }

  /**
   * Discover available functions
   */
  async discoverFunctions() {
    if (!this.client) {
      throw new Error("Sidecar not started. Call start() first.");
    }
    return this.client.discoverFunctions();
  }

  /**
   * Invoke a Forst function
   */
  async invoke(packageName: string, functionName: string, args: any = {}) {
    if (!this.client) {
      throw new Error("Sidecar not started. Call start() first.");
    }
    return this.client.invokeFunction(packageName, functionName, args);
  }

  /**
   * Invoke a Forst function with streaming
   */
  async invokeStreaming(
    packageName: string,
    functionName: string,
    args: any = {},
    onResult?: (result: any) => void
  ) {
    if (!this.client) {
      throw new Error("Sidecar not started. Call start() first.");
    }
    return this.client.invokeStreaming(
      packageName,
      functionName,
      args,
      onResult
    );
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

/**
 * Express.js middleware for easy integration
 */
export function createExpressMiddleware(sidecar: ForstSidecar) {
  return async (req: any, res: any, next: any) => {
    // Add sidecar to request object for easy access
    req.forst = sidecar;
    next();
  };
}

/**
 * Auto-start function for zero-config usage
 */
export async function autoStart(
  config?: Partial<ForstConfig>
): Promise<ForstSidecar> {
  const sidecar = new ForstSidecar(config);
  await sidecar.start();

  // Handle graceful shutdown
  process.on("SIGINT", async () => {
    logger.info("\n🛑 Received SIGINT, shutting down gracefully...");
    await sidecar.stop();
    process.exit(0);
  });

  process.on("SIGTERM", async () => {
    logger.info("\n🛑 Received SIGTERM, shutting down gracefully...");
    await sidecar.stop();
    process.exit(0);
  });

  return sidecar;
}

// Default export for convenience
export default ForstSidecar;
