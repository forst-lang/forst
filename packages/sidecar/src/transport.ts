import { Effect } from "effect";
import {
  SIDECAR_PACKAGE_VERSION,
  SIDECAR_VERSION_HTTP_HEADER,
} from "./constants";

/** Callable fetch shape for injection and tests. */
export type FetchImpl = (
  input: string | URL | Request,
  init?: RequestInit
) => Promise<Response>;

export interface HttpInvokeTransportConfig {
  baseUrl: string;
  timeout?: number;
  fetchFn?: FetchImpl;
  extraHeaders?: Record<string, string>;
}

/** Low-level HTTP transport for dev-server RPC paths (e.g. `/invoke`, `/health`). */
export interface InvokeTransport {
  request(
    endpoint: string,
    init?: RequestInit
  ): Effect.Effect<Response, Error>;
}

const defaultFetch: FetchImpl = (input, init) => fetch(input, init);

/** Plain HTTP {@link InvokeTransport} with sidecar version header and per-request timeout. */
export function createHttpInvokeTransport(
  config: HttpInvokeTransportConfig
): InvokeTransport {
  const baseUrl = config.baseUrl.replace(/\/$/, "");
  const timeoutMs = config.timeout ?? 30_000;
  const fetchFn = config.fetchFn ?? defaultFetch;
  const defaultHeaders: Record<string, string> = {
    [SIDECAR_VERSION_HTTP_HEADER]: SIDECAR_PACKAGE_VERSION,
    "Content-Type": "application/json",
    ...config.extraHeaders,
  };

  return {
    request(endpoint, init) {
      const url = `${baseUrl}${endpoint}`;
      return Effect.tryPromise({
        try: () =>
          fetchFn(url, {
            ...init,
            headers: {
              ...defaultHeaders,
              ...(init?.headers as Record<string, string> | undefined),
            },
            signal: init?.signal ?? AbortSignal.timeout(timeoutMs),
          }),
        catch: (error) =>
          error instanceof Error ? error : new Error(String(error)),
      });
    },
  };
}
