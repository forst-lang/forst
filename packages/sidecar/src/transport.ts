import { Effect } from "effect";
import {
  SIDECAR_PACKAGE_VERSION,
  SIDECAR_VERSION_HTTP_HEADER,
} from "./constants";
import {
  computeInvokeProof,
  normalizeHeaders,
  stripReservedHeaders,
} from "./invoke-auth";
import { fetchInvokeChallenge } from "./invoke-challenge";
import { readInvokeReadyAuth, readInvokeReadySocketPath } from "./invoke-ready";
import { isUnixSocketSupported, requestOverUnixSocket } from "./unix-transport";

/** Callable fetch shape for injection and tests. */
export type FetchImpl = (
  input: string | URL | Request,
  init?: RequestInit
) => Promise<Response>;

/** Shared invoke transport options. */
export type HttpInvokeTransportShared = {
  timeout?: number;
  fetchFn?: FetchImpl;
  extraHeaders?: Record<string, string>;
  /**
   * Project root for `.forst/invoke.ready` metadata (URL, socketPath, generation)
   * and `.forst/invoke.token` when auth uses the token-file profile.
   */
  boundaryRoot?: string;
};

/**
 * Invoke RPC auth profile. Modes are mutually exclusive.
 *
 * - `authDisabled`: legacy escape hatch (`FORST_INVOKE_AUTH=off`, tests)
 * - `resolveAuth`: memory handoff after spawn (Strong profile)
 * - token file via `boundaryRoot`: connect / TCP fallback (Standard profile)
 */
export type HttpInvokeTransportAuthConfig =
  | { authDisabled: true; resolveAuth?: never }
  | {
      authDisabled?: false;
      resolveAuth: () =>
        | { token: Uint8Array; generation: number }
        | undefined;
    }
  | {
      authDisabled?: false;
      resolveAuth?: never;
    };

/**
 * TCP base URL resolution. Use either a static `baseUrl` or a `resolveBaseUrl`
 * callback (with optional `baseUrl` fallback), not both as co-primary sources.
 */
export type HttpInvokeTransportUrlConfig =
  | { baseUrl: string; resolveBaseUrl?: never }
  | { resolveBaseUrl: () => string | undefined; baseUrl?: string }
  | { baseUrl?: never; resolveBaseUrl?: never };

/**
 * Unix socket dial target. An explicit `socketPath` overrides ready-file discovery
 * under `boundaryRoot`.
 */
export type HttpInvokeTransportSocketConfig =
  | { socketPath: string }
  | { socketPath?: never };

/**
 * Configuration for {@link createHttpInvokeTransport}: base URL resolution
 * (static, dynamic, or ready-file), auth profile, optional UDS, timeouts, and headers.
 */
export type HttpInvokeTransportConfig = HttpInvokeTransportShared &
  HttpInvokeTransportAuthConfig &
  HttpInvokeTransportUrlConfig &
  HttpInvokeTransportSocketConfig;

/** Builds a transport config that satisfies the mutually exclusive unions. */
export function buildHttpInvokeTransportConfig(
  input: HttpInvokeTransportShared & {
    baseUrl?: string;
    resolveBaseUrl?: () => string | undefined;
    resolveAuth?: () =>
      | { token: Uint8Array; generation: number }
      | undefined;
    socketPath?: string;
    authDisabled?: boolean;
  }
): HttpInvokeTransportConfig {
  const shared: HttpInvokeTransportShared = {
    timeout: input.timeout,
    fetchFn: input.fetchFn,
    extraHeaders: input.extraHeaders,
    boundaryRoot: input.boundaryRoot,
  };

  const auth: HttpInvokeTransportAuthConfig = input.authDisabled
    ? { authDisabled: true }
    : input.resolveAuth
      ? { resolveAuth: input.resolveAuth }
      : {};

  const url: HttpInvokeTransportUrlConfig = input.resolveBaseUrl
    ? { resolveBaseUrl: input.resolveBaseUrl, baseUrl: input.baseUrl }
    : input.baseUrl
      ? { baseUrl: input.baseUrl }
      : {};

  const socket: HttpInvokeTransportSocketConfig = input.socketPath
    ? { socketPath: input.socketPath }
    : {};

  return { ...shared, ...auth, ...url, ...socket };
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
  const timeoutMs = config.timeout ?? 30_000;
  const fetchFn = config.fetchFn ?? defaultFetch;
  const transportHeaders: Record<string, string> = {
    [SIDECAR_VERSION_HTTP_HEADER]: SIDECAR_PACKAGE_VERSION,
    "Content-Type": "application/json",
  };

  const resolveBaseUrl = (): string | undefined => {
    const raw =
      "resolveBaseUrl" in config && config.resolveBaseUrl
        ? config.resolveBaseUrl()
        : "baseUrl" in config
          ? config.baseUrl
          : undefined;
    if (!raw) {
      return undefined;
    }
    return raw.replace(/\/$/, "");
  };

  const resolveSocketPath = (): string | undefined => {
    if ("socketPath" in config && config.socketPath) {
      return config.socketPath;
    }
    return readInvokeReadySocketPath(config.boundaryRoot);
  };

  const resolveAuthState = ():
    | { token: Uint8Array; generation: number }
    | undefined => {
    if ("resolveAuth" in config && config.resolveAuth) {
      return config.resolveAuth();
    }
    const auth = readInvokeReadyAuth(config.boundaryRoot);
    if (!auth) {
      return undefined;
    }
    return { token: auth.token, generation: auth.generation };
  };

  const selfTransport: InvokeTransport = {
    request(endpoint, init) {
      return Effect.tryPromise({
        try: () => performRequest(endpoint, init),
        catch: (error) =>
          error instanceof Error ? error : new Error(String(error)),
      });
    },
  };

  async function performRequest(
    endpoint: string,
    init?: RequestInit
  ): Promise<Response> {
    const userHeaders = stripReservedHeaders({
      ...normalizeHeaders(init?.headers),
      ...normalizeHeaders(config.extraHeaders),
    });
    const headers: Record<string, string> = {
      ...userHeaders,
      ...transportHeaders,
    };

    const needsAuth =
      !("authDisabled" in config && config.authDisabled) &&
      endpoint !== "/health" &&
      endpoint !== "/invoke/challenge";

    if (needsAuth) {
      const auth = resolveAuthState();
      if (!auth) {
        throw new Error("invoke transport: missing auth state");
      }
      const challenge = await Effect.runPromise(
        fetchInvokeChallenge(selfTransport)
      );
      headers["X-Forst-Invoke-Nonce"] = challenge.nonce;
      headers["X-Forst-Invoke-Generation"] = String(auth.generation);
      headers["X-Forst-Invoke-Proof"] = computeInvokeProof(
        auth.token,
        auth.generation,
        challenge.nonce
      );
    }

    const socketPath = resolveSocketPath();
    if (socketPath && isUnixSocketSupported()) {
      return requestOverUnixSocket(socketPath, endpoint, {
        ...init,
        headers,
        signal: init?.signal ?? AbortSignal.timeout(timeoutMs),
      });
    }

    const baseUrl = resolveBaseUrl();
    if (!baseUrl) {
      throw new Error("invoke transport: missing baseUrl");
    }
    const url = `${baseUrl}${endpoint}`;
    return fetchFn(url, {
      ...init,
      headers,
      signal: init?.signal ?? AbortSignal.timeout(timeoutMs),
    });
  }

  return selfTransport;
}
