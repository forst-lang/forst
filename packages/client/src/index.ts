import { ForstSidecar, type ForstSidecar as ForstSidecarType } from "@forst/sidecar";
import { clientLogger } from "./logger";
import {
  createInvokeClient,
  getDefaultInvokeClient,
  resetDefaultInvokeClientForTest,
  type ForstInvokeClient,
  type ForstInvokeClientConfig,
} from "./invoke-client";

export type ForstClientConfig = ForstInvokeClientConfig;

export {
  createInvokeClient,
  getDefaultInvokeClient,
  resetDefaultInvokeClientForTest,
  type ForstInvokeClient,
  type ForstInvokeClientConfig,
};

/**
 * Main Forst client class that provides a Prisma-like experience
 */
export class ForstClient {
  private sidecar: ForstSidecar;
  private invokeClient: ForstInvokeClient;
  private packages: Map<string, any> = new Map();
  private isManaged: boolean = false;
  private cleanupHandler?: () => void;

  constructor(config?: Partial<ForstClientConfig>) {
    const defaultConfig: ForstClientConfig = {
      baseUrl:
        process.env.FORST_BASE_URL ||
        process.env.FORST_DEV_URL ||
        "http://127.0.0.1:8081",
      timeout: 30000,
      retries: 3,
      mode: "development",
      port: 8080,
      host: "localhost",
      logLevel: "info",
      ...config,
    };

    this.invokeClient = createInvokeClient(defaultConfig);

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
        sidecarRuntime: defaultConfig.sidecarRuntime,
        devServerUrl: defaultConfig.devServerUrl ?? defaultConfig.baseUrl,
      });
      this.isManaged = true;
    }

    return new Proxy(this, {
      get(target, prop) {
        if (prop in target) {
          return (target as any)[prop];
        }
        if (typeof prop === "string") {
          return target.getPackage(prop);
        }
        return undefined;
      },
    }) as ForstClient;
  }

  async start(): Promise<void> {
    if (this.isManaged) {
      await this.sidecar.start();
    }
  }

  async stop(): Promise<void> {
    if (this.isManaged) {
      await this.sidecar.stop();
    }
  }

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

    process.on("SIGINT", cleanup);
    process.on("SIGTERM", cleanup);
    process.on("SIGQUIT", cleanup);
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

  cleanup(): void {
    if (this.cleanupHandler) {
      this.cleanupHandler();
    }
  }

  async healthCheck(): Promise<boolean> {
    return this.invokeClient.healthCheck();
  }

  getServerInfo(): any {
    return this.sidecar.getServerInfo();
  }

  async discoverFunctions(): Promise<any[]> {
    return this.sidecar.discoverFunctions();
  }

  getPackage(packageName: string): any {
    if (!this.packages.has(packageName)) {
      const packageProxy = new Proxy(
        {},
        {
          get: (_target, prop) => {
            if (typeof prop === "string") {
              return async (...args: any[]) => {
                return this.invoke(packageName, prop, args);
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

  async invoke(
    packageName: string,
    functionName: string,
    args?: any[]
  ): Promise<any> {
    const response = await this.invokeClient.invokeFunction(
      packageName,
      functionName,
      args ?? []
    );
    if (!response.success) {
      throw new Error(
        response.error || `${packageName}.${functionName} failed`
      );
    }
    return response.result;
  }
}

export default ForstClient;

export type {
  InvokeRequest,
  InvokeResponse,
  FunctionInfo,
} from "@forst/sidecar";
