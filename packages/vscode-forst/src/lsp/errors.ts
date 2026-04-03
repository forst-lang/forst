/**
 * Typed failures from the HTTP JSON-RPC bridge so activation code can surface
 * actionable messages instead of treating every fault as an opaque string.
 */

/** Base marker for failures originating in the Forst HTTP LSP client layer. */
export class LspClientError extends Error {
  constructor(message: string, options?: ErrorOptions) {
    super(message, options);
    this.name = "LspClientError";
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

/** Raised when the POST endpoint returns a non-success status (server or proxy mismatch). */
export class LspHttpResponseError extends LspClientError {
  constructor(
    readonly status: number,
    readonly bodySnippet: string
  ) {
    super(`LSP HTTP ${status}: ${bodySnippet}`);
    this.name = "LspHttpResponseError";
  }
}

/** Raised when the body cannot be parsed as JSON-RPC (version skew or truncated response). */
export class LspMalformedResponseError extends LspClientError {
  constructor(
    readonly rawBodySnippet: string,
    cause: unknown
  ) {
    super(`Invalid JSON from LSP: ${rawBodySnippet}`, { cause });
    this.name = "LspMalformedResponseError";
  }
}

/** Raised when the response carries a JSON-RPC `error` object (method not found, invalid params, etc.). */
export class LspJsonRpcResponseError extends LspClientError {
  constructor(
    readonly rpcCode: number,
    rpcMessage: string
  ) {
    super(
      rpcMessage.trim().length > 0 ? rpcMessage : `LSP error ${rpcCode}`
    );
    this.name = "LspJsonRpcResponseError";
  }
}

/** Raised when `/health` never succeeds before the deadline (binary not started, wrong port, firewall). */
export class LspHealthTimeoutError extends LspClientError {
  constructor(
    readonly healthUrl: string,
    readonly lastError: string | undefined
  ) {
    super(
      `Forst LSP did not respond on ${healthUrl} (${lastError ?? "unknown"})`
    );
    this.name = "LspHealthTimeoutError";
  }
}
