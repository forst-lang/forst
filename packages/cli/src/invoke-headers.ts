/**
 * Shared header helpers for invoke auth and Unix transport.
 * Kept separate so `unix-transport` and `invoke-auth` do not import each other.
 */

/** Reserved invoke auth header names (lowercase). Callers must not set these. */
export const RESERVED_INVOKE_HEADERS = [
  "x-forst-invoke-proof",
  "x-forst-invoke-generation",
  "x-forst-invoke-nonce",
  "x-forst-invoke-token",
] as const;

/** Header bag accepted without requiring DOM lib typings. */
export type InvokeHeadersInit =
  | Headers
  | Record<string, string>
  | Array<[string, string]>;

/**
 * Normalizes `Headers` / tuples / records into a plain string map.
 */
export function normalizeHeaders(
  headers: InvokeHeadersInit | undefined
): Record<string, string> {
  if (!headers) {
    return {};
  }
  if (headers instanceof Headers) {
    const out: Record<string, string> = {};
    headers.forEach((value, key) => {
      out[key] = value;
    });
    return out;
  }
  if (Array.isArray(headers)) {
    return Object.fromEntries(headers);
  }
  return { ...headers };
}

/**
 * Removes reserved Forst invoke auth headers (case-insensitive) from a header map.
 */
export function stripReservedHeaders(
  headers: Record<string, string>
): Record<string, string> {
  const out: Record<string, string> = {};
  for (const [key, value] of Object.entries(headers)) {
    const lower = key.toLowerCase();
    if (
      RESERVED_INVOKE_HEADERS.includes(
        lower as (typeof RESERVED_INVOKE_HEADERS)[number]
      )
    ) {
      continue;
    }
    out[key] = value;
  }
  return out;
}
