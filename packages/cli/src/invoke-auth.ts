import { createHmac } from "node:crypto";

/**
 * Domain separator baked into every invoke HMAC message.
 * Must stay in sync with Go `invokeProofVersion` in `forst/internal/invokeserver`.
 */
export const INVOKE_PROOF_VERSION = "forst-invoke-v1";

/**
 * Formats the MAC input string: `version|generation|nonce`.
 *
 * @param generation Live auth generation from the invoke server.
 * @param nonce Single-use challenge nonce from `GET /invoke/challenge`.
 */
export function invokeProofMessage(generation: number, nonce: string): string {
  return `${INVOKE_PROOF_VERSION}|${generation}|${nonce}`;
}

/**
 * Computes the HMAC-SHA256 invoke proof as unpadded base64url.
 * Matches Go `computeInvokeProof` / `encodeInvokeProof`.
 *
 * @param token Raw 32-byte invoke secret (from handoff or `invoke.token`).
 * @param generation Live auth generation bound into the MAC.
 * @param nonce Single-use challenge nonce.
 * @returns Proof string for the `X-Forst-Invoke-Proof` header.
 */
export function computeInvokeProof(
  token: Uint8Array,
  generation: number,
  nonce: string
): string {
  return createHmac("sha256", token)
    .update(invokeProofMessage(generation, nonce))
    .digest("base64url");
}

/** Live invoke secret and generation used to build proof headers. */
export interface InvokeAuthState {
  /** Raw HMAC key bytes (not base64). */
  token: Uint8Array;
  /** Monotonic generation from `invoke.ready` / challenge payload. */
  generation: number;
}

/** Payload from `GET /invoke/challenge` after unwrapping the invoke JSON envelope. */
export interface InvokeChallenge {
  /** Single-use nonce; must be sent once then discarded. */
  nonce: string;
  /** Server generation at challenge time (informational; proof uses caller auth generation). */
  generation: number;
  /** RFC3339 expiry for the nonce. */
  expiresAt: string;
}

/** Injectable fetch for tests. */
type FetchLike = (
  input: string | URL | Request,
  init?: RequestInit
) => Promise<Response>;

/**
 * Fetches a single-use nonce from `GET /invoke/challenge`.
 * The challenge endpoint does not require a proof header (peer/backoff still apply).
 *
 * @param baseUrl Invoke server base URL (trailing slash optional).
 * @param fetchFn Optional fetch implementation (defaults to global `fetch`).
 * @returns Parsed challenge with `nonce`, `generation`, and `expiresAt`.
 * @throws When the HTTP status is not OK or the body lacks a nonce.
 */
export async function fetchInvokeChallenge(
  baseUrl: string,
  fetchFn: FetchLike = fetch
): Promise<InvokeChallenge> {
  const url = `${baseUrl.replace(/\/$/, "")}/invoke/challenge`;
  const response = await fetchFn(url, { method: "GET" });
  if (!response.ok) {
    throw new Error(`invoke challenge failed: HTTP ${response.status}`);
  }
  const payload = (await response.json()) as {
    success?: boolean;
    result?: InvokeChallenge | string;
  };
  const raw = payload.result;
  const parsed =
    typeof raw === "string"
      ? (JSON.parse(raw) as InvokeChallenge)
      : (raw as InvokeChallenge | undefined);
  if (!parsed?.nonce) {
    throw new Error("invoke challenge missing nonce");
  }
  return parsed;
}

/**
 * Builds reserved proof headers for one authenticated RPC request.
 *
 * @param auth Token and generation used to MAC the nonce.
 * @param nonce Fresh nonce from {@link fetchInvokeChallenge}.
 * @returns `X-Forst-Invoke-Nonce`, `X-Forst-Invoke-Generation`, and `X-Forst-Invoke-Proof`.
 */
export function buildInvokeAuthHeaders(
  auth: InvokeAuthState,
  nonce: string
): Record<string, string> {
  return {
    "X-Forst-Invoke-Nonce": nonce,
    "X-Forst-Invoke-Generation": String(auth.generation),
    "X-Forst-Invoke-Proof": computeInvokeProof(
      auth.token,
      auth.generation,
      nonce
    ),
  };
}

/**
 * Sends `POST /invoke` with a fresh challenge-response proof.
 *
 * Use after `startForstInvokeServer` when auth is enabled
 * (`FORST_INVOKE_AUTH` is not `off`). Pair with `readInvokeReadyAuth`
 * to load the token from `.forst/invoke.token`.
 *
 * @param baseUrl Invoke server base URL.
 * @param body JSON-serializable invoke request body.
 * @param auth Live token and generation.
 * @param fetchFn Optional fetch implementation.
 * @returns The raw `Response` from `POST /invoke`.
 */
export async function fetchAuthenticatedInvoke(
  baseUrl: string,
  body: unknown,
  auth: InvokeAuthState,
  fetchFn: FetchLike = fetch
): Promise<Response> {
  const challenge = await fetchInvokeChallenge(baseUrl, fetchFn);
  return fetchFn(`${baseUrl.replace(/\/$/, "")}/invoke`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...buildInvokeAuthHeaders(auth, challenge.nonce),
    },
    body: JSON.stringify(body),
  });
}
