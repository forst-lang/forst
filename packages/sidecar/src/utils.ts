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
  getExpectedCompilerBinaryPath,
  getCompilerArtifactDownloadUrlForCurrentPlatform,
  fetchLatestCompilerReleaseVersion,
  CompilerBinaryDownloadHttpFailure as CliCompilerBinaryDownloadHttpFailure,
  CompilerBinaryDownloadFailed as CliCompilerBinaryDownloadFailed,
  CompilerBinaryChecksumMismatch as CliCompilerBinaryChecksumMismatch,
  CompilerBinaryNotFound as CliCompilerBinaryNotFound,
  UnsupportedArchitecture as CliUnsupportedArchitecture,
  UnsupportedOperatingSystem as CliUnsupportedOperatingSystem,
} from "@forst/cli";
import { CompilerInfo, ForstConfig } from "./types";
import {
  CompilerBinaryDownloadFailed,
  CompilerBinaryDownloadHttpFailure,
  CompilerNotFound,
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
  /** Maps @forst/cli errors to sidecar errors (same rules as resolveCompilerBinary). */
  private static mapCliError(e: unknown): never {
    if (e instanceof CliCompilerBinaryDownloadHttpFailure) {
      throw new CompilerBinaryDownloadHttpFailure(e.status, e.statusText);
    }
    if (e instanceof CliCompilerBinaryDownloadFailed) {
      throw new CompilerBinaryDownloadFailed(e.message, { cause: e.cause });
    }
    if (e instanceof CliCompilerBinaryChecksumMismatch) {
      throw new CompilerBinaryDownloadFailed(e.message, { cause: e });
    }
    if (e instanceof CliCompilerBinaryNotFound) {
      throw new CompilerNotFound(e.message, { cause: e });
    }
    if (e instanceof CliUnsupportedArchitecture) {
      throw new UnsupportedArchitecture(arch());
    }
    if (e instanceof CliUnsupportedOperatingSystem) {
      throw new UnsupportedOperatingSystem(platform());
    }
    throw e;
  }

  private static async resolveCompilerBinary(
    allowDownload: boolean
  ): Promise<string> {
    try {
      return await resolveForstBinary({ allowDownload });
    } catch (e) {
      this.mapCliError(e);
    }
  }

  /**
   * Latest published compiler semver from GitHub `releases/latest`.
   */
  static async getLatestVersion(): Promise<string> {
    try {
      return await fetchLatestCompilerReleaseVersion();
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      throw new LatestCompilerReleaseUnavailable(
        `Failed to fetch latest Forst version: ${msg}`,
        { cause: e instanceof Error ? e : undefined }
      );
    }
  }

  /**
   * Get the platform-specific compiler binary name
   */
  static getCompilerBinaryName(): string {
    try {
      return getCompilerArtifactName(platform(), arch());
    } catch (e) {
      this.mapCliError(e);
    }
  }

  /**
   * Get the download URL for the compiler binary (pinned to @forst/cli / compiler semver).
   */
  static getCompilerDownloadUrl(): string {
    try {
      return getCompilerArtifactDownloadUrlForCurrentPlatform();
    } catch (e) {
      this.mapCliError(e);
    }
  }

  /**
   * Path where @forst/cli caches the binary for this package version (may not exist yet).
   */
  static getCompilerLocalPath(): string {
    try {
      return getExpectedCompilerBinaryPath(getCliPackageVersion());
    } catch (e) {
      this.mapCliError(e);
    }
  }

  /**
   * Download the compiler binary from GitHub releases (when missing).
   */
  static async downloadCompiler(): Promise<string> {
    utilsLogger.info("Resolving Forst compiler via @forst/cli (download allowed)");
    try {
      const path = await this.resolveCompilerBinary(true);
      utilsLogger.info(`✅ Forst compiler available at: ${path}`);
      return path;
    } catch (error) {
      utilsLogger.error("❌ Failed to download compiler:", error);
      throw error;
    }
  }

  /**
   * Ensure the compiler binary is available locally.
   * @param allowDownload When false (default), only FORST_BINARY or an existing cache hit is used.
   */
  static async ensureCompiler(allowDownload = false): Promise<string> {
    const localPath = await this.resolveCompilerBinary(allowDownload);
    utilsLogger.info(`✅ Using Forst compiler at: ${localPath}`);
    return localPath;
  }

  /**
   * Get compiler information
   */
  static async getCompilerInfo(allowDownload = false): Promise<CompilerInfo> {
    const version = getCliPackageVersion();
    const binaryPath = await this.resolveCompilerBinary(allowDownload);

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
    let stdout: string;
    let exitCode: number;
    try {
      const r = await this.executeForstCommand(forstPath, ["version"], {});
      stdout = r.stdout;
      exitCode = r.exitCode;
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      utilsLogger.warn(
        `Could not spawn forst version (skipping local semver): ${msg}`
      );
      return "";
    }
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
