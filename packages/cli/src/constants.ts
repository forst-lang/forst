/** GitHub Releases base (compiler tags are vX.Y.Z). */
export const COMPILER_RELEASES_BASE =
  "https://github.com/forst-lang/forst/releases";

/**
 * Pass as the only argument to the npm `forst` shim (`npx forst --forst-cli-info`) to print
 * wrapper semver, resolved binary path, and `forst version` — without forwarding to other subcommands.
 */
export const FORST_CLI_INFO_FLAG = "--forst-cli-info";
