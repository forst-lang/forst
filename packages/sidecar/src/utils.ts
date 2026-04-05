import { spawn, SpawnOptions } from "node:child_process";
import {
  existsSync,
  mkdirSync,
  writeFileSync,
  readFileSync,
} from "node:fs";
import { join, dirname, resolve as resolvePath } from "node:path";
import { platform, arch } from "node:os";
import {
  resolveForstBinary,
  getCompilerArtifactName,
  getCliPackageVersion,
  buildCompilerArtifactDownloadUrl,
  getExpectedCompilerBinaryPath,
  CompilerBinaryDownloadHttpFailure as CliCompilerBinaryDownloadHttpFailure,
  CompilerBinaryDownloadFailed as CliCompilerBinaryDownloadFailed,
  UnsupportedArchitecture as CliUnsupportedArchitecture,
  UnsupportedOperatingSystem as CliUnsupportedOperatingSystem,
} from "@forst/cli";
import { CompilerInfo, ForstConfig } from "./types";
import {
  CompilerBinaryDownloadFailed,
  CompilerBinaryDownloadHttpFailure,
  LatestCompilerReleaseUnavailable,
  SpawnedProcessExitedNonZero,
  UnsupportedArchitecture,
  UnsupportedOperatingSystem,
} from "./errors";
import { utilsLogger } from "./logger";

/**
 * Compiler download, PATH resolution, filesystem helpers, and generic process utilities for the sidecar.
 */
export class ForstUtils {
  private static async resolveCompilerBinary(): Promise<string> {
    try {
      return await resolveForstBinary();
    } catch (e) {
      if (e instanceof CliCompilerBinaryDownloadHttpFailure) {
        throw new CompilerBinaryDownloadHttpFailure(e.status, e.statusText);
      }
      if (e instanceof CliCompilerBinaryDownloadFailed) {
        throw new CompilerBinaryDownloadFailed(e.message, { cause: e.cause });
      }
      if (e instanceof CliUnsupportedArchitecture) {
        throw new UnsupportedArchitecture(arch());
      }
      if (e instanceof CliUnsupportedOperatingSystem) {
        throw new UnsupportedOperatingSystem(platform());
      }
      throw e;
    }
  }

  /**
   * Get the latest version from GitHub releases
   */
  static async getLatestVersion(): Promise<string> {
    try {
      const response = await fetch(
        "https://api.github.com/repos/forst-lang/forst/releases/latest"
      );
      const data = (await response.json()) as { tag_name: string };
      return data.tag_name.replace("v", ""); // Remove 'v' prefix
    } catch (error) {
      throw new LatestCompilerReleaseUnavailable("Failed to fetch latest Forst version", {
        cause: error instanceof Error ? error : undefined,
      });
    }
  }

  /**
   * Get the platform-specific compiler binary name
   */
  static getCompilerBinaryName(): string {
    return getCompilerArtifactName(platform(), arch());
  }

  /**
   * Get the download URL for the compiler binary (pinned to @forst/cli / compiler semver).
   */
  static async getCompilerDownloadUrl(): Promise<string> {
    const version = getCliPackageVersion();
    const binaryName = this.getCompilerBinaryName();
    return buildCompilerArtifactDownloadUrl(version, binaryName);
  }

  /**
   * Path where @forst/cli caches the binary for this package version (may not exist yet).
   */
  static getCompilerLocalPath(): string {
    return getExpectedCompilerBinaryPath(getCliPackageVersion());
  }

  /**
   * Download the compiler binary from GitHub releases
   */
  static async downloadCompiler(): Promise<string> {
    utilsLogger.info("Resolving Forst compiler via @forst/cli");
    try {
      const path = await this.resolveCompilerBinary();
      utilsLogger.info(`✅ Forst compiler available at: ${path}`);
      return path;
    } catch (error) {
      utilsLogger.error("❌ Failed to download compiler:", error);
      throw error;
    }
  }

  /**
   * Ensure the compiler binary is available locally
   */
  static async ensureCompiler(): Promise<string> {
    const localPath = await this.resolveCompilerBinary();
    utilsLogger.info(`✅ Using Forst compiler at: ${localPath}`);
    return localPath;
  }

  /**
   * Get compiler information
   */
  static async getCompilerInfo(): Promise<CompilerInfo> {
    const version = getCliPackageVersion();
    const binaryPath = await this.resolveCompilerBinary();

    return {
      version,
      path: binaryPath,
      platform: platform(),
      arch: arch(),
    };
  }

  /**
   * Parses the first line of `forst version` stdout (`forst <version> <commit> <date>`).
   */
  static parseForstVersionStdout(stdout: string): string {
    const line = stdout.trim().split("\n")[0] ?? "";
    const m = /^forst\s+(\S+)/i.exec(line);
    return m?.[1] ?? "";
  }

  /**
   * Runs `forst version` on the given binary and returns the semver string, or "" on failure.
   */
  static async getLocalBinaryVersion(forstPath: string): Promise<string> {
    const { stdout, exitCode } = await this.executeForstCommand(
      forstPath,
      ["version"],
      {}
    );
    if (exitCode !== 0) {
      return "";
    }
    return this.parseForstVersionStdout(stdout);
  }

  /**
   * Spawn a Forst process using the resolved compiler binary path.
   */
  static spawnForstProcess(
    forstPath: string,
    args: string[],
    options: SpawnOptions = {}
  ): ReturnType<typeof spawn> {
    return spawn(forstPath, args, {
      stdio: ["pipe", "pipe", "pipe"],
      ...options,
    });
  }

  /**
   * Execute a Forst command and return the result
   */
  static async executeForstCommand(
    forstPath: string,
    args: string[],
    options: SpawnOptions = {}
  ): Promise<{ stdout: string; stderr: string; exitCode: number }> {
    return new Promise((resolve, reject) => {
      const process = this.spawnForstProcess(forstPath, args, options);

      let stdout = "";
      let stderr = "";

      process.stdout?.on("data", (data) => {
        stdout += data.toString();
      });

      process.stderr?.on("data", (data) => {
        stderr += data.toString();
      });

      process.on("close", (code) => {
        resolve({
          stdout,
          stderr,
          exitCode: code || 0,
        });
      });

      process.on("error", (error) => {
        reject(error);
      });
    });
  }

  /**
   * Detect Forst files in the project
   */
  static async detectForstFiles(config: ForstConfig): Promise<string[]> {
    const forstDir = config.forstDir || "./forst";
    const fullPath = resolvePath(forstDir);

    if (!existsSync(fullPath)) {
      return [];
    }

    // Simple glob pattern for .ft files
    const { glob } = await import("glob");
    const pattern = join(fullPath, "**/*.ft");

    try {
      const files = await glob(pattern, { nodir: true });
      return files;
    } catch (error) {
      utilsLogger.warn("Failed to detect Forst files:", error);
      return [];
    }
  }

  /**
   * Create default Forst configuration
   */
  static createDefaultConfig(): ForstConfig {
    return {
      mode:
        process.env.NODE_ENV === "production" ? "production" : "development",
      forstDir: "./forst",
      outputDir: "./dist/forst",
      port: 8080,
      host: "localhost",
      logLevel: "info",
      transports: {
        development: {
          mode: "http",
          http: {
            port: 8080,
            cors: true,
            healthCheck: "/health",
          },
        },
        production: {
          mode: "http",
          http: {
            port: 8080,
            cors: true,
            healthCheck: "/health",
          },
        },
        testing: {
          mode: "http",
          http: {
            port: 0, // Random port
            cors: false,
          },
        },
      },
    };
  }

  /**
   * Run a command and return the result
   */
  static async runCommand(
    command: string,
    args: string[] = [],
    options: SpawnOptions = {}
  ): Promise<{ stdout: string; stderr: string; code: number }> {
    return new Promise((resolve, reject) => {
      const child = spawn(command, args, {
        stdio: ["pipe", "pipe", "pipe"],
        ...options,
      });

      let stdout = "";
      let stderr = "";

      child.stdout?.on("data", (data) => {
        stdout += data.toString();
      });

      child.stderr?.on("data", (data) => {
        stderr += data.toString();
      });

      child.on("close", (code) => {
        if (code === 0) {
          resolve({ stdout, stderr, code });
        } else {
          reject(new SpawnedProcessExitedNonZero(code ?? -1, stderr));
        }
      });

      child.on("error", (error) => {
        reject(error);
      });
    });
  }

  /**
   * Ensure directory exists
   */
  static ensureDir(path: string): void {
    if (!existsSync(path)) {
      mkdirSync(path, { recursive: true });
    }
  }

  /**
   * Write file with error handling
   */
  static writeFile(path: string, content: string): void {
    this.ensureDir(dirname(path));
    writeFileSync(path, content, "utf8");
  }

  /**
   * Read file with error handling
   */
  static readFile(path: string): string {
    return readFileSync(path, "utf8");
  }

  /**
   * Generate a unique identifier
   */
  static generateId(): string {
    return (
      Math.random().toString(36).substring(2, 15) +
      Math.random().toString(36).substring(2, 15)
    );
  }
}
