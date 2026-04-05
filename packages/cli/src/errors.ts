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

/** HTTP download failed (non-2xx). */
export class CompilerBinaryDownloadHttpFailure extends Error {
  readonly status: number;
  readonly statusText: string;

  constructor(status: number, statusText: string) {
    super(`Download failed: HTTP ${status} ${statusText}`);
    this.name = "CompilerBinaryDownloadHttpFailure";
    this.status = status;
    this.statusText = statusText;
  }
}

/** Download or install failed. */
export class CompilerBinaryDownloadFailed extends Error {
  constructor(message: string, options?: { cause?: unknown }) {
    super(message, options);
    this.name = "CompilerBinaryDownloadFailed";
  }
}

/** Downloaded bytes do not match the sha256 from GitHub release metadata. */
export class CompilerBinaryChecksumMismatch extends Error {
  readonly expectedHex: string;
  readonly actualHex: string;

  constructor(expectedHex: string, actualHex: string) {
    super(
      `Forst compiler checksum mismatch (possible corrupt download). Expected sha256 ${expectedHex}, got ${actualHex}`
    );
    this.name = "CompilerBinaryChecksumMismatch";
    this.expectedHex = expectedHex;
    this.actualHex = actualHex;
  }
}
