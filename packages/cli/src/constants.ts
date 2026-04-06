/** GitHub Releases base (compiler tags are vX.Y.Z). */
export const COMPILER_RELEASES_BASE =
  "https://github.com/forst-lang/forst/releases";

/**
 * Pass as the only argument to the npm `forst` shim (`npx forst --forst-cli-info`) to print
 * wrapper semver, resolved binary path, and `forst version` — without forwarding to other subcommands.
 */
export const FORST_CLI_INFO_FLAG = "--forst-cli-info";

/**
 * Pass as the only argument to print `@forst/cli` semver and the resolved compiler’s `forst version`
 * first line — compact, for scripts and `--version` conventions (does not forward to the native binary).
 */
export const FORST_CLI_VERSION_FLAGS = ["--version", "-V"] as const;

/**
 * Pass as the only argument to print embedded Go module metadata for the resolved `forst` binary via
 * `go version -m` (requires a Go toolchain on PATH). Does not forward to the native binary.
 */
export const FORST_CLI_GO_BUILDINFO_FLAG = "--forst-go-buildinfo";
