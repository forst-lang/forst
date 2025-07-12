import { ForstSidecar } from "@forst/sidecar";
import { clientLogger } from "./logger";

export interface ForstClientConfig {
  baseUrl?: string;
  timeout?: number;
  retries?: number;
  mode?: "development" | "production";
  // Dev server management options
  port?: number;
  host?: string;
  logLevel?: "info" | "debug" | "warn" | "error";
  rootDir?: string;
  // Custom sidecar instance
  customSidecar?: ForstSidecar;
}

/**
 * Package namespace proxy that provides type-safe function calls
 */
class PackageProxy {
  private sidecar: ForstSidecar;
  private packageName: string;

  constructor(sidecar: ForstSidecar, packageName: string) {
    this.sidecar = sidecar;
    this.packageName = packageName;
  }

  /**
   * Dynamic function call that invokes the underlying Forst function
   */
  async callFunction(functionName: string, ...args: any[]): Promise<any> {
    const response = await this.sidecar.invoke(
      this.packageName,
      functionName,
      ...args
    );
    if (!response.success) {
      throw new Error(
        response.error || `${this.packageName}.${functionName} failed`
      );
    }
    return response.result;
  }
}

/**
 * Main Forst client class that provides a Prisma-like experience
 */
export class ForstClient {
  private sidecar: ForstSidecar;
  private packages: Map<string, any> = new Map();
  private isManaged: boolean = false;
  private cleanupHandler?: () => void;

  constructor(config?: Partial<ForstClientConfig>) {
    const defaultConfig: ForstClientConfig = {
      baseUrl: process.env.FORST_BASE_URL || "http://localhost:8080",
      timeout: 30000,
      retries: 3,
      mode: "development",
      port: 8080,
      host: "localhost",
      logLevel: "info",
      ...config,
    };

    // If a custom sidecar is provided, use it
    if (defaultConfig.customSidecar) {
      clientLogger.warn(
        "⚠️  Using custom sidecar - this is not recommended for normal use"
      );
      this.sidecar = defaultConfig.customSidecar;
      this.isManaged = false;
    } else {
      this.sidecar = new ForstSidecar({
        mode: defaultConfig.mode || "development",
        port: defaultConfig.port || 8080,
        host: defaultConfig.host || "localhost",
        logLevel: defaultConfig.logLevel || "info",
        rootDir: defaultConfig.rootDir || process.cwd(),
      });
      this.isManaged = true;
    }

    // Create a proxy for dynamic package access
    return new Proxy(this, {
      get(target, prop) {
        // If it's a property of the client, return it
        if (prop in target) {
          return (target as any)[prop];
        }

        // If it's a string, treat it as a package name
        if (typeof prop === "string") {
          return target.getPackage(prop);
        }

        return undefined;
      },
    }) as ForstClient;
  }

  /**
   * Start the dev server (only if managed)
   */
  async start(): Promise<void> {
    if (this.isManaged) {
      await this.sidecar.start();
    }
  }

  /**
   * Stop the dev server (only if managed)
   */
  async stop(): Promise<void> {
    if (this.isManaged) {
      await this.sidecar.stop();
    }
  }

  /**
   * Set up cleanup handlers for managed mode
   */
  setupCleanup(): void {
    if (!this.isManaged) return;

    const cleanup = async () => {
      try {
        await this.sidecar.stop();
      } catch (error) {
        clientLogger.error("Failed to stop Forst server:", error);
      }
      process.exit(0);
    };

    // Handle various interrupt signals
    process.on("SIGINT", cleanup);
    process.on("SIGTERM", cleanup);
    process.on("SIGQUIT", cleanup);

    // Handle uncaught exceptions and unhandled rejections
    process.on("uncaughtException", (error) => {
      clientLogger.error("Uncaught exception:", error);
      cleanup();
    });

    process.on("unhandledRejection", (reason, promise) => {
      clientLogger.error("Unhandled rejection at:", promise, "reason:", reason);
      cleanup();
    });

    this.cleanupHandler = () => {
      process.off("SIGINT", cleanup);
      process.off("SIGTERM", cleanup);
      process.off("SIGQUIT", cleanup);
      process.off("uncaughtException", cleanup);
      process.off("unhandledRejection", cleanup);
    };
  }

  /**
   * Clean up signal handlers
   */
  cleanup(): void {
    if (this.cleanupHandler) {
      this.cleanupHandler();
    }
  }

  /**
   * Health check
   */
  async healthCheck(): Promise<boolean> {
    return this.sidecar.healthCheck();
  }

  /**
   * Get server info
   */
  getServerInfo(): any {
    return this.sidecar.getServerInfo();
  }

  /**
   * Discover available functions
   */
  async discoverFunctions(): Promise<any[]> {
    return this.sidecar.discoverFunctions();
  }

  /**
   * Get a package namespace with type-safe function calls
   */
  getPackage(packageName: string): any {
    if (!this.packages.has(packageName)) {
      // Create a dynamic package proxy that supports function calls
      const packageProxy = new Proxy(
        {},
        {
          get: (target, prop) => {
            if (typeof prop === "string") {
              return async (...args: any[]) => {
                clientLogger.debug(
                  `🚀 Package proxy calling ${packageName}.${prop} with args:`,
                  args
                );
                const response = await this.sidecar.invoke(
                  packageName,
                  prop,
                  ...args
                );
                clientLogger.debug(
                  `📦 Package proxy received response for ${packageName}.${prop}:`,
                  response
                );
                if (!response.success) {
                  clientLogger.error(
                    `❌ Package proxy call failed for ${packageName}.${prop}:`,
                    response.error
                  );
                  throw new Error(
                    response.error || `${packageName}.${prop} failed`
                  );
                }
                clientLogger.debug(
                  `✅ Package proxy call successful for ${packageName}.${prop}:`,
                  response.result
                );
                return response.result;
              };
            }
            return undefined;
          },
        }
      );

      this.packages.set(packageName, packageProxy);
    }

    return this.packages.get(packageName);
  }

  /**
   * Direct invoke method for compatibility
   */
  async invoke(
    packageName: string,
    functionName: string,
    args?: any
  ): Promise<any> {
    clientLogger.debug(
      `🚀 Client invoking ${packageName}.${functionName} with args:`,
      args
    );
    const response = await this.sidecar.invoke(packageName, functionName, args);
    clientLogger.debug(
      `📦 Client received response for ${packageName}.${functionName}:`,
      response
    );
    if (!response.success) {
      clientLogger.error(
        `❌ Client invoke failed for ${packageName}.${functionName}:`,
        response.error
      );
      throw new Error(
        response.error || `${packageName}.${functionName} failed`
      );
    }
    clientLogger.debug(
      `✅ Client invoke successful for ${packageName}.${functionName}:`,
      response.result
    );
    return response.result;
  }

  /**
   * Dynamic property access for package namespaces
   * This enables the Prisma-like syntax: forst.package1.function1()
   */
  get [Symbol.iterator]() {
    return this.packages[Symbol.iterator];
  }
}

// Export the client class
export default ForstClient;

// Re-export types from sidecar for convenience
export type {
  InvokeRequest,
  InvokeResponse,
  FunctionInfo,
} from "@forst/sidecar";
