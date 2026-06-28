/** Snapshot of `forst.*` workspace settings the LSP layer and spawn logic need in one place. */
export interface ForstExtensionConfig {
  forstPath: string;
  port: number;
  logLevel: string;
  autoStart: boolean;
  /**
   * When true (default), use @forst/cli to resolve FORST_BINARY, cache, or download the compiler.
   * When false, use workspace `bin/forst` / PATH only.
   */
  downloadCompiler: boolean;
  /**
   * When true (default) and downloads are enabled, compare the bundled @forst/cli semver with
   * GitHub's latest release and use the higher version for the cache path (may fetch a newer binary).
   */
  preferLatestCompilerRelease: boolean;
}
