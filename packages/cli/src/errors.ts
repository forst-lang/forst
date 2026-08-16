import type { DigestLookupReason } from "./github-release.js";

/** Raised when the platform is not supported for prebuilt binaries. */
export class UnsupportedOperatingSystem extends Error {
  constructor(message: string) {
    super(message);
    this.name = "UnsupportedOperatingSystem";
  }
}

/** Raised when the CPU architecture is not supported for prebuilt binaries. */
export class UnsupportedArchitecture extends Error {
  constructor(message: string) {
    super(message);
    this.name = "UnsupportedArchitecture";
  }
}

/** Version, artifact, and URL for compiler binary download failures. */
export interface CompilerBinaryDownloadContext {
  version: string;
  artifact: string;
  url: string;
}

function compilerBinaryDownloadHttpFailureMessage(
  status: number,
  statusText: string,
  context?: CompilerBinaryDownloadContext
): string {
  const base = `Download failed: HTTP ${status} ${statusText}`;
  if (!context) {
    return `${base}. Run npx forst --forst-cli-info for diagnostics.`;
  }

  let message =
    `${base} for ${context.artifact} (release v${context.version}). ` +
    `URL: ${context.url}. `;
  if (status === 404) {
    message +=
      "The GitHub release or asset may not exist yet. Upgrade @forst/cli, check forst.compilerRelease, or set FORST_BINARY. ";
  }
  message += "Run npx forst --forst-cli-info for diagnostics.";
  return message;
}

/** HTTP download failed (non-2xx). */
export class CompilerBinaryDownloadHttpFailure extends Error {
  readonly status: number;
  readonly statusText: string;
  readonly version?: string;
  readonly artifact?: string;
  readonly url?: string;

  constructor(
    status: number,
    statusText: string,
    context?: CompilerBinaryDownloadContext
  ) {
    super(compilerBinaryDownloadHttpFailureMessage(status, statusText, context));
    this.name = "CompilerBinaryDownloadHttpFailure";
    this.status = status;
    this.statusText = statusText;
    this.version = context?.version;
    this.artifact = context?.artifact;
    this.url = context?.url;
  }
}

/** Download or install failed. */
export class CompilerBinaryDownloadFailed extends Error {
  constructor(message: string, options?: { cause?: unknown }) {
    super(message, options);
    this.name = "CompilerBinaryDownloadFailed";
  }
}

/** No local compiler and downloading was disabled (see {@link resolveForstBinary} `allowDownload`). */
export class CompilerBinaryNotFound extends Error {
  constructor(message: string, options?: { cause?: unknown }) {
    super(message, options);
    this.name = "CompilerBinaryNotFound";
  }
}

const STRICT_DIGEST_SUFFIX =
  "Set FORST_CLI_VERIFY=0 to allow unverified downloads, or point FORST_BINARY at a trusted compiler.";

function strictDigestUnavailableMessage(
  version: string,
  artifact: string,
  reason: DigestLookupReason
): string {
  switch (reason) {
    case "release_not_found":
      return (
        `FORST_CLI_VERIFY=strict requires a sha256 digest, but GitHub release v${version} does not exist. ` +
        `Check forst.compilerRelease / the release tag. ${STRICT_DIGEST_SUFFIX}`
      );
    case "asset_not_found":
      return (
        `FORST_CLI_VERIFY=strict requires a sha256 digest, but release v${version} has no asset ${artifact}. ` +
        `Wrong compiler version or platform. ${STRICT_DIGEST_SUFFIX}`
      );
    case "digest_missing":
      return (
        `FORST_CLI_VERIFY=strict requires a sha256 digest, but GitHub returned no sha256 for ${artifact} on v${version}. ` +
        STRICT_DIGEST_SUFFIX
      );
  }
}

/**
 * Raised when `FORST_CLI_VERIFY=strict` (or `1`) and GitHub provides no digest
 * to verify against.
 */
export class CompilerBinaryDigestUnavailable extends Error {
  readonly version: string;
  readonly artifact: string;
  readonly reason: DigestLookupReason;

  constructor(version: string, artifact: string, reason: DigestLookupReason) {
    super(strictDigestUnavailableMessage(version, artifact, reason));
    this.name = "CompilerBinaryDigestUnavailable";
    this.version = version;
    this.artifact = artifact;
    this.reason = reason;
  }
}

/** Version/artifact pair threaded through checksum errors so failures name which download they belong to. */
export interface CompilerBinaryChecksumContext {
  version: string;
  artifact: string;
}

/** Downloaded bytes do not match the sha256 from GitHub release metadata. */
export class CompilerBinaryChecksumMismatch extends Error {
  readonly expectedHex: string;
  readonly actualHex: string;
  readonly version?: string;
  readonly artifact?: string;

  constructor(
    expectedHex: string,
    actualHex: string,
    context?: CompilerBinaryChecksumContext
  ) {
    const version = context?.version ?? "unknown";
    const artifact = context?.artifact ?? "unknown";
    super(
      `Forst compiler SHA-256 verification failed for ${artifact} (release v${version}). ` +
        `The downloaded bytes do not match GitHub release metadata. ` +
        `This may indicate a corrupted download, a network proxy altering the file, or a supply-chain attack. ` +
        `Do not run this binary — delete the cached compiler file and retry, or set FORST_BINARY to a trusted compiler. ` +
        `Expected sha256 ${expectedHex}, got ${actualHex}.`
    );
    this.name = "CompilerBinaryChecksumMismatch";
    this.expectedHex = expectedHex;
    this.actualHex = actualHex;
    this.version = context?.version;
    this.artifact = context?.artifact;
  }
}

/** Context carried by invoke-server lifecycle failures from `@forst/cli/invoke`. */
export interface ForstInvokeServerErrorContext {
  mode?: "auto" | "dev" | "embedded";
  port?: number;
  argv?: string[];
  stderrTail?: string;
  baseUrl?: string;
  exitCode?: number | null;
}

/** Timed out waiting for GET /health after spawning an invoke server. */
export class ForstInvokeServerStartTimeout extends Error {
  readonly mode?: ForstInvokeServerErrorContext["mode"];
  readonly port?: number;
  readonly argv?: string[];
  readonly stderrTail?: string;
  readonly baseUrl?: string;

  constructor(message: string, ctx: ForstInvokeServerErrorContext = {}) {
    super(message);
    this.name = "ForstInvokeServerStartTimeout";
    this.mode = ctx.mode;
    this.port = ctx.port;
    this.argv = ctx.argv;
    this.stderrTail = ctx.stderrTail;
    this.baseUrl = ctx.baseUrl;
  }
}

/** Child process exited before the invoke server became healthy. */
export class ForstInvokeServerExitedEarly extends Error {
  readonly mode?: ForstInvokeServerErrorContext["mode"];
  readonly port?: number;
  readonly argv?: string[];
  readonly stderrTail?: string;
  readonly exitCode?: number | null;

  constructor(message: string, ctx: ForstInvokeServerErrorContext = {}) {
    super(message);
    this.name = "ForstInvokeServerExitedEarly";
    this.mode = ctx.mode;
    this.port = ctx.port;
    this.argv = ctx.argv;
    this.stderrTail = ctx.stderrTail;
    this.exitCode = ctx.exitCode;
  }
}

/** Could not reach an existing invoke server on the attach path. */
export class ForstInvokeServerUnreachable extends Error {
  readonly baseUrl?: string;
  readonly port?: number;

  constructor(message: string, ctx: ForstInvokeServerErrorContext = {}) {
    super(message);
    this.name = "ForstInvokeServerUnreachable";
    this.baseUrl = ctx.baseUrl;
    this.port = ctx.port;
  }
}
