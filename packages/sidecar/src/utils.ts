import { spawn, SpawnOptions } from "node:child_process";
import {
  existsSync,
  mkdirSync,
  writeFileSync,
  readFileSync,
  chmodSync,
} from "node:fs";
import { join, dirname, resolve } from "node:path";
import { platform, arch } from "node:os";
import { CompilerInfo, ForstConfig } from "./types";
import { utilsLogger } from "./logger";

export class ForstUtils {
  private static readonly COMPILER_BASE_URL =
    "https://github.com/forst-lang/forst/releases";

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
      throw new Error("Failed to fetch latest Forst version");
    }
  }

  /**
   * Get the platform-specific compiler binary name
   */
  static getCompilerBinaryName(): string {
    const os = platform();
    const architecture = arch();
    // Validate OS and architecture
    if (!["darwin", "linux", "win32"].includes(os)) {
      throw new Error(`Unsupported operating system: ${os}`);
    }

    if (!["arm64", "x64"].includes(architecture)) {
      throw new Error(`Unsupported architecture: ${architecture}`);
    }

    // Map OS to platform name in binary
    const platformName = os === "win32" ? "windows" : os;

    const ext = os === "win32" ? ".exe" : "";
    const archString = architecture === "x64" ? "amd64" : architecture;

    return `forst-${platformName}-${archString}${ext}`;
  }

  /**
   * Get the download URL for the compiler binary
   */
  static async getCompilerDownloadUrl(): Promise<string> {
    const version = await this.getLatestVersion();
    const binaryName = this.getCompilerBinaryName();
    return `${this.COMPILER_BASE_URL}/download/v${version}/${binaryName}`;
  }

  /**
   * Get the local path where the compiler binary should be stored
   */
  static getCompilerLocalPath(): string {
    const binaryName = this.getCompilerBinaryName();
    const nodeModulesPath = resolve(process.cwd(), "node_modules");
    const forstBinPath = join(nodeModulesPath, ".bin", "forst");

    // Ensure the .bin directory exists
    const binDir = dirname(forstBinPath);
    if (!existsSync(binDir)) {
      mkdirSync(binDir, { recursive: true });
    }

    return forstBinPath;
  }

  /**
   * Download the compiler binary from GitHub releases
   */
  static async downloadCompiler(): Promise<string> {
    const downloadUrl = await this.getCompilerDownloadUrl();
    const localPath = this.getCompilerLocalPath();

    utilsLogger.info(`Downloading Forst compiler from: ${downloadUrl}`);
    utilsLogger.info(`Saving to: ${localPath}`);

    try {
      const response = await fetch(downloadUrl);
      if (!response.ok) {
        throw new Error(
          `Failed to download compiler: ${response.status} ${response.statusText}`
        );
      }

      const arrayBuffer = await response.arrayBuffer();
      const buffer = Buffer.from(arrayBuffer);

      writeFileSync(localPath, buffer);
      chmodSync(localPath, 0o755); // Make executable

      utilsLogger.info("✅ Forst compiler downloaded successfully");
      return localPath;
    } catch (error) {
      utilsLogger.error("❌ Failed to download compiler:", error);
      throw error;
    }
  }

  /**
   * Ensure the compiler binary is available locally
   */
  static async ensureCompiler(): Promise<string> {
    const localPath = this.getCompilerLocalPath();

    // Check if binary already exists
    if (existsSync(localPath)) {
      utilsLogger.info(`✅ Using existing Forst compiler at: ${localPath}`);
      return localPath;
    }

    // Download the binary
    return await this.downloadCompiler();
  }

  /**
   * Get compiler information
   */
  static async getCompilerInfo(): Promise<CompilerInfo> {
    const version = await this.getLatestVersion();
    const binaryPath = await this.ensureCompiler();

    return {
      version,
      path: binaryPath,
      platform: platform(),
      arch: arch(),
    };
  }

  /**
   * Spawn a Forst process with the given arguments
   */
  static spawnForstProcess(
    args: string[],
    options: SpawnOptions = {}
  ): ReturnType<typeof spawn> {
    return spawn("forst", args, {
      stdio: ["pipe", "pipe", "pipe"],
      ...options,
    });
  }

  /**
   * Execute a Forst command and return the result
   */
  static async executeForstCommand(
    args: string[],
    options: SpawnOptions = {}
  ): Promise<{ stdout: string; stderr: string; exitCode: number }> {
    return new Promise((resolve, reject) => {
      const process = this.spawnForstProcess(args, options);

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
    const fullPath = resolve(forstDir);

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
          reject(new Error(`Command failed with code ${code}: ${stderr}`));
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
