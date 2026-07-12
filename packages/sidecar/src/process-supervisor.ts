import { spawn, type ChildProcess } from "node:child_process";
import {
  DevServerChildProcessNotResponding,
  DevServerChildShutdownTimeout,
  DevServerHealthCheckHttpFailure,
  DevServerStartupTimeout,
} from "./errors";
import { serverLogger, forstLogger } from "./logger";

export type ForstDevSpawnPlan = {
  args: string[];
  cwd: string;
};

export type ProcessSupervisorStatus =
  | "stopped"
  | "starting"
  | "running"
  | "error";

/**
 * Spawns and supervises the `forst dev` child process (start, health wait, graceful stop).
 */
export class ProcessSupervisor {
  private process: ChildProcess | null = null;
  private status: ProcessSupervisorStatus = "stopped";

  constructor(
    private readonly forstPath: string,
    private readonly host: string,
    private readonly port: number
  ) {}

  get child(): ChildProcess | null {
    return this.process;
  }

  get processStatus(): ProcessSupervisorStatus {
    return this.status;
  }

  setProcessStatus(status: ProcessSupervisorStatus): void {
    this.status = status;
  }

  async start(plan: ForstDevSpawnPlan): Promise<void> {
    if (this.process && !this.process.killed) {
      return;
    }

    this.status = "starting";
    const { args, cwd } = plan;

    serverLogger.info(
      `Starting Forst server with: ${this.forstPath} ${args.join(" ")}`
    );

    this.process = spawn(this.forstPath, args, {
      stdio: ["pipe", "pipe", "pipe"],
      cwd,
    });

    this.process.on("error", (error) => {
      serverLogger.error("Forst server process error:", error);
      this.status = "error";
    });

    this.process.on("exit", (code, signal) => {
      serverLogger.info(
        `Forst server process exited with code ${code}, signal ${signal}`
      );
      this.status = "stopped";
    });

    this.process.stdout?.on("data", (data) => {
      const output = data.toString();
      const trimmedOutput = output.trim();
      if (trimmedOutput) {
        forstLogger.info(`[Forst] ${trimmedOutput}`);
      }
      if (output.includes("HTTP server listening")) {
        serverLogger.debug("Server ready detected from stdout");
        this.status = "running";
      }
    });

    this.process.stderr?.on("data", (data) => {
      const error = data.toString();
      const trimmedError = error.trim();
      if (trimmedError) {
        const logMethods = {
          "level=debug": forstLogger.debug,
          "level=info": forstLogger.info,
          "level=warn": forstLogger.warn,
          "level=error": forstLogger.error,
        } as const;
        const [, logMethod] = Object.entries(logMethods).find(([level]) =>
          trimmedError.includes(level)
        ) || [null, forstLogger.info];
        logMethod(trimmedError);
      }
      if (error.includes("HTTP server listening")) {
        serverLogger.debug("Server ready detected from stderr");
        this.status = "running";
      }
    });

    serverLogger.debug("Waiting for server to be ready...");
    await this.waitForReady();
    serverLogger.debug("Server ready check completed");
  }

  async stop(): Promise<void> {
    if (!this.process) {
      this.status = "stopped";
      return;
    }

    const proc = this.process;
    this.process = null;

    try {
      proc.kill("SIGTERM");
      await new Promise<void>((resolve, reject) => {
        const timeout = setTimeout(() => {
          reject(new DevServerChildShutdownTimeout());
        }, 5000);

        proc.once("exit", (code, signal) => {
          clearTimeout(timeout);
          forstLogger.debug(
            `Forst server process exited gracefully with code ${code}, signal ${signal}`
          );
          resolve();
        });
      });
    } catch (error) {
      serverLogger.warn("Graceful shutdown failed, forcing kill:", error);
      try {
        proc.kill("SIGKILL");
        await new Promise<void>((resolve) => {
          setTimeout(() => {
            forstLogger.info("Forst server process force killed");
            resolve();
          }, 1000);
        });
      } catch (killError) {
        serverLogger.error("Failed to force kill process:", killError);
      }
    } finally {
      this.status = "stopped";
    }
  }

  async waitForReady(): Promise<void> {
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        serverLogger.error("⏰ Server startup timeout after 10 seconds");
        reject(new DevServerStartupTimeout());
      }, 10_000);

      setTimeout(async () => {
        try {
          const healthUrl = `http://${this.host}:${this.port}/health`;
          serverLogger.debug(`🏥 Checking server health at: ${healthUrl}`);
          const response = await fetch(healthUrl);
          serverLogger.debug(
            `🏥 Health check response status: ${response.status}`
          );

          if (response.ok) {
            const healthData = await response.text();
            serverLogger.debug(`🏥 Health check response body: ${healthData}`);
            this.status = "running";
            clearTimeout(timeout);
            serverLogger.info("✅ Server is ready and responding");
            resolve();
            return;
          }

          const errorText = await response.text();
          serverLogger.error(
            `❌ Server health check failed with status ${response.status}: ${errorText}`
          );
          clearTimeout(timeout);
          reject(
            new DevServerHealthCheckHttpFailure(response.status, errorText)
          );
        } catch (error) {
          serverLogger.error("❌ Health check request failed:", error);
          if (this.process && !this.process.killed) {
            serverLogger.warn(
              "⚠️  Health check failed but process is alive, marking as running"
            );
            this.status = "running";
            clearTimeout(timeout);
            resolve();
          } else {
            serverLogger.error(
              "❌ Health check failed and process is not alive"
            );
            clearTimeout(timeout);
            reject(new DevServerChildProcessNotResponding());
          }
        }
      }, 2000);
    });
  }
}
