/** Snapshot of `forst.*` workspace settings the LSP layer and spawn logic need in one place. */
export interface ForstExtensionConfig {
  /**
   * `forst.compiler.path`: path to the `forst` executable.
   * Empty/default `"forst"` means resolve via PATH, workspace `bin/forst`, or `@forst/cli`.
   */
  forstPath: string;
  /**
   * `forst.go.path`: optional path to the `go` binary or its directory.
   * Prepended on PATH when spawning `forst lsp` so `packages.Load` can find the toolchain.
   * Empty means heuristics only (Homebrew, Program Files, Scoop, `GOROOT`/`GOPATH`, etc.).
   */
  goPath: string;
  /**
   * `forst.lsp.port`: TCP port for the Forst LSP HTTP server (`POST /` JSON-RPC).
   * Default `8081`.
   */
  port: number;
  /**
   * `forst.lsp.logLevel`: log level passed to `forst lsp -log-level`
   * (`trace` | `debug` | `info` | `warn` | `error`). Default `info`.
   */
  logLevel: string;
  /**
   * `forst.lsp.autoStart`: when true, spawn `forst lsp` as a child on activation;
   * when false, connect to an already-running server on `port`. Default `true`.
   */
  autoStart: boolean;
  /**
   * `forst.compiler.download`: when true, use `@forst/cli` to resolve `FORST_BINARY`,
   * cache, or download the compiler when no workspace `bin/forst` / explicit path is found.
   * When false, use workspace `bin/forst` / PATH only. Default `true`.
   */
  downloadCompiler: boolean;
  /**
   * `forst.compiler.preferLatestRelease`: when true (and downloads are enabled), compare the
   * bundled `@forst/cli` semver with GitHub's latest release and use the higher version for
   * the cache path (may fetch a newer binary). Default `true`.
   */
  preferLatestCompilerRelease: boolean;
}
