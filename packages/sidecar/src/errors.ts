/** Base class for failures thrown by `@forst/sidecar`. */
export class ForstError extends Error {
  constructor(message: string, options?: ErrorOptions) {
    super(message, options);
    this.name = "ForstError";
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/** Sidecar APIs used before `ForstSidecar.start()` completed. */
export class SidecarNotStarted extends ForstError {
  constructor(message = "Sidecar not started. Call start() first.") {
    super(message);
    this.name = "SidecarNotStarted";
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/** No Forst compiler binary could be resolved (PATH / download). */
export class CompilerNotFound extends ForstError {
  constructor(
    message = "Forst compiler not found. Please ensure the Forst compiler is installed and available in your PATH.",
    options?: ErrorOptions
  ) {
    super(message, options);
    this.name = "CompilerNotFound";
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/** `GET /functions` JSON envelope had `success: false`. */
export class DevServerFunctionsRejected extends ForstError {
  readonly serverError: string | undefined;

  constructor(serverError: string | undefined) {
    super(
      `Failed to discover functions${serverError ? `: ${serverError}` : ""}`
    );
    this.name = "DevServerFunctionsRejected";
    this.serverError = serverError;
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/** `invoke("pkg.fn")` must use exactly one dot-separated package and name. */
export class InvalidFunctionNameFormat extends ForstError {
  readonly input: string;

  constructor(input: string) {
    super(
      `Invalid function name format: ${input}. Expected format: package.function`
    );
    this.name = "InvalidFunctionNameFormat";
    this.input = input;
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/**
 * Fetch to `forst dev` returned a non-success HTTP status (before JSON parsing),
 * or streaming `POST /invoke` was not OK.
 */
export class DevServerHttpFailure extends ForstError {
  readonly status: number;
  readonly responseText: string;

  constructor(status: number, responseText: string) {
    super(`HTTP ${status}: ${responseText}`);
    this.name = "DevServerHttpFailure";
    this.status = status;
    this.responseText = responseText;
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/** Streaming `POST /invoke` succeeded at HTTP level but returned no body to read. */
export class DevServerStreamingInvokeNoResponseBody extends ForstError {
  constructor(message = "No response body available for streaming") {
    super(message);
    this.name = "DevServerStreamingInvokeNoResponseBody";
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/** `GET /types` JSON envelope had `success: false`. */
export class DevServerTypesRejected extends ForstError {
  readonly serverError: string | undefined;

  constructor(serverError: string | undefined) {
    super(serverError || "Failed to fetch generated TypeScript types");
    this.name = "DevServerTypesRejected";
    this.serverError = serverError;
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/** `GET /types` JSON was OK but omitted `output` (merged TS string). */
export class DevServerTypesOutputMissing extends ForstError {
  constructor(message = "Missing types output in /types response") {
    super(message);
    this.name = "DevServerTypesOutputMissing";
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/** All retries to reach `forst dev` over HTTP failed (see `cause`). */
export class DevServerRequestRetriesExhausted extends ForstError {
  readonly lastError: Error | null;

  constructor(lastError: Error | null) {
    super("Request failed after all retries", lastError ? { cause: lastError } : undefined);
    this.name = "DevServerRequestRetriesExhausted";
    this.lastError = lastError;
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/** Could not read latest compiler release metadata (e.g. GitHub API or network). */
export class LatestCompilerReleaseUnavailable extends ForstError {
  constructor(message = "Failed to fetch latest Forst version", options?: ErrorOptions) {
    super(message, options);
    this.name = "LatestCompilerReleaseUnavailable";
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

export class UnsupportedOperatingSystem extends ForstError {
  readonly os: string;

  constructor(os: string) {
    super(`Unsupported operating system: ${os}`);
    this.name = "UnsupportedOperatingSystem";
    this.os = os;
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

export class UnsupportedArchitecture extends ForstError {
  readonly architecture: string;

  constructor(architecture: string) {
    super(`Unsupported architecture: ${architecture}`);
    this.name = "UnsupportedArchitecture";
    this.architecture = architecture;
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/** Compiler binary download URL returned a non-success HTTP status. */
export class CompilerBinaryDownloadHttpFailure extends ForstError {
  readonly status: number;
  readonly statusText: string;

  constructor(status: number, statusText: string) {
    super(`Failed to download compiler: ${status} ${statusText}`);
    this.name = "CompilerBinaryDownloadHttpFailure";
    this.status = status;
    this.statusText = statusText;
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/** Compiler binary download failed for a non-HTTP reason (I/O, TLS, etc.). */
export class CompilerBinaryDownloadFailed extends ForstError {
  constructor(message: string, options?: ErrorOptions) {
    super(message, options);
    this.name = "CompilerBinaryDownloadFailed";
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/** Child process exited non-zero from `ForstUtils.runCommand`. */
export class SpawnedProcessExitedNonZero extends ForstError {
  readonly exitCode: number;
  readonly stderr: string;

  constructor(exitCode: number, stderr: string) {
    super(`Command failed with code ${exitCode}: ${stderr}`);
    this.name = "SpawnedProcessExitedNonZero";
    this.exitCode = exitCode;
    this.stderr = stderr;
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/** `forst dev` child did not exit after SIGTERM within the shutdown window. */
export class DevServerChildShutdownTimeout extends ForstError {
  constructor(
    message = "Forst dev child process did not terminate gracefully within timeout"
  ) {
    super(message);
    this.name = "DevServerChildShutdownTimeout";
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/** `forst dev` did not become reachable (startup wait exceeded). */
export class DevServerStartupTimeout extends ForstError {
  constructor(message = "Forst dev server startup timeout") {
    super(message);
    this.name = "DevServerStartupTimeout";
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/** Local health check to `forst dev` returned a non-success HTTP status. */
export class DevServerHealthCheckHttpFailure extends ForstError {
  readonly status: number;
  readonly responseText: string;

  constructor(status: number, responseText: string) {
    super(`Forst dev health check failed: HTTP ${status}: ${responseText}`);
    this.name = "DevServerHealthCheckHttpFailure";
    this.status = status;
    this.responseText = responseText;
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/** Health request failed and the `forst dev` child process is no longer alive. */
export class DevServerChildProcessNotResponding extends ForstError {
  constructor(
    message = "Forst dev server process not responding (child not alive)"
  ) {
    super(message);
    this.name = "DevServerChildProcessNotResponding";
    Object.setPrototypeOf(this, new.target.prototype);
  }
}
