/**
 * Redacts sensitive keys and truncates large values for safe debug logging.
 *
 * Intended for `logger.debug` only — never log raw request/response bodies in production
 * without sanitization.
 */

// --- Types ------------------------------------------------------------------

/** Controls recursion depth and truncation when cloning values for logs. */
export interface SanitizeOptions {
  readonly maxDepth: number;
  readonly maxStringLen: number;
  readonly maxArrayItems: number;
  readonly maxObjectKeys: number;
}

/** Default limits suitable for verbose debug output. */
export const DEFAULT_SANITIZE_OPTIONS: SanitizeOptions = {
  maxDepth: 8,
  maxStringLen: 256,
  maxArrayItems: 24,
  maxObjectKeys: 32,
} as const;

const LOG_LIMITS = {
  /** Skip parsing JSON bodies larger than this (bytes). */
  maxBodyBytesForParse: 64 * 1024,
  maxOpaqueBodyPreviewChars: 200,
  maxErrorPreviewChars: 200,
  maxHeaderValueChars: 80,
} as const;

// --- Key / header redaction -------------------------------------------------

/** Exact and substring patterns for object keys (case-insensitive). */
const SENSITIVE_KEY_PATTERN =
  /^(authorization|password|token|secret|cookie|apikey|api[_-]?key|bearer|set-cookie)$/i;

function isSensitiveKey(key: string): boolean {
  if (SENSITIVE_KEY_PATTERN.test(key)) return true;
  const lower = key.toLowerCase();
  return lower.includes("authorization") || lower.includes("password");
}

function shouldRedactHeaderName(name: string): boolean {
  return isSensitiveKey(name) || /auth|token|cookie|secret/i.test(name);
}

function truncate(str: string, maxLen: number): string {
  if (str.length <= maxLen) return str;
  return `${str.slice(0, maxLen)}… (${str.length} chars)`;
}

type HeadersForEach = {
  forEach(callback: (value: string, name: string) => void): void;
};

function isHeadersLike(value: object): value is HeadersForEach {
  return typeof (value as HeadersForEach).forEach === "function";
}

function trySanitizeHeadersLike(value: object): Record<string, string> | null {
  if (!isHeadersLike(value)) return null;
  try {
    const out: Record<string, string> = {};
    value.forEach((val, key) => {
      out[key] = shouldRedactHeaderName(key)
        ? "[redacted]"
        : truncate(val, LOG_LIMITS.maxHeaderValueChars);
    });
    return out;
  } catch {
    return null;
  }
}

// --- Core sanitization ------------------------------------------------------

/**
 * Returns a JSON-serializable snapshot of `value` with secrets redacted and large
 * structures truncated.
 */
export function sanitizePayload(
  value: unknown,
  depth = 0,
  options: SanitizeOptions = DEFAULT_SANITIZE_OPTIONS
): unknown {
  if (value === null || value === undefined) {
    return value;
  }
  if (depth > options.maxDepth) {
    return "[max depth]";
  }

  switch (typeof value) {
    case "string":
      return truncate(value, options.maxStringLen);
    case "number":
    case "boolean":
      return value;
    case "bigint":
      return `${value.toString()}n`;
    case "function":
      return "[Function]";
    case "symbol":
      return value.toString();
    case "undefined":
      return value;
    default:
      break;
  }

  if (typeof value !== "object") {
    return String(value);
  }

  if (Array.isArray(value)) {
    return sanitizeArray(value, depth, options);
  }

  const headersRecord = trySanitizeHeadersLike(value);
  if (headersRecord !== null) {
    return headersRecord;
  }

  return sanitizePlainObject(value as Record<string, unknown>, depth, options);
}

function sanitizeArray(
  arr: unknown[],
  depth: number,
  options: SanitizeOptions
): unknown[] {
  const cap = options.maxArrayItems;
  const slice = arr.slice(0, cap).map((item) =>
    sanitizePayload(item, depth + 1, options)
  );
  if (arr.length <= cap) {
    return slice;
  }
  return [
    ...slice,
    `… (${arr.length - cap} more items)`,
  ];
}

function sanitizePlainObject(
  obj: Record<string, unknown>,
  depth: number,
  options: SanitizeOptions
): Record<string, unknown> {
  const keys = Object.keys(obj);
  const cap = options.maxObjectKeys;
  const selectedKeys = keys.slice(0, cap);
  const out: Record<string, unknown> = {};

  for (const key of selectedKeys) {
    if (isSensitiveKey(key)) {
      out[key] = "[redacted]";
      continue;
    }
    const child = obj[key];
    if (key.toLowerCase() === "headers" && child !== null && typeof child === "object") {
      out[key] = sanitizePayload(child, depth + 1, options);
      continue;
    }
    out[key] = sanitizePayload(child, depth + 1, options);
  }

  if (keys.length > cap) {
    out["…"] = `${keys.length - cap} more keys`;
  }
  return out;
}

// --- HTTP / invoke helpers --------------------------------------------------

/**
 * Parses a request body string when small enough; otherwise returns byte size only.
 * JSON bodies are passed through {@link sanitizePayload}.
 */
export function sanitizeRequestBodyString(body: string | undefined): unknown {
  if (body === undefined) {
    return undefined;
  }
  if (body.length > LOG_LIMITS.maxBodyBytesForParse) {
    return {
      bytes: body.length,
      note: "body too large for log preview",
    };
  }
  try {
    const parsed: unknown = JSON.parse(body);
    return sanitizePayload(parsed);
  } catch {
    return {
      bytes: body.length,
      preview: truncate(body, LOG_LIMITS.maxOpaqueBodyPreviewChars),
    };
  }
}

/** Fields safe to log for an RPC-style response (result is sanitized). */
export function invokeResponseLogFields(res: {
  success: boolean;
  error?: string;
  result?: unknown;
}): Record<string, unknown> {
  const { error, result, success } = res;
  return {
    success,
    error:
      error === undefined
        ? undefined
        : truncate(error, LOG_LIMITS.maxErrorPreviewChars),
    result: sanitizePayload(result),
  };
}

/** Metadata + sanitized args for invoke / streaming calls. */
export function invokeRequestLogFields(req: {
  package: string;
  function: string;
  args: unknown;
  streaming?: boolean;
}): Record<string, unknown> {
  const { argsSize, argCount } = measureArgs(req.args);
  return {
    package: req.package,
    function: req.function,
    streaming: req.streaming ?? false,
    argCount,
    argsPayloadBytes: argsSize,
    args: sanitizePayload(req.args),
  };
}

function measureArgs(args: unknown): { argsSize: number; argCount: number } {
  try {
    const serialized = JSON.stringify(args);
    const argsSize = serialized.length;
    let argCount: number;
    if (Array.isArray(args)) {
      argCount = args.length;
    } else if (args !== null && typeof args === "object") {
      argCount = Object.keys(args as object).length;
    } else {
      argCount = args === undefined ? 0 : 1;
    }
    return { argsSize, argCount };
  } catch {
    return { argsSize: 0, argCount: -1 };
  }
}
