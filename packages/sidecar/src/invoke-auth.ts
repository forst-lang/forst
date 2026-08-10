import { createHmac } from "node:crypto";

export const INVOKE_PROOF_VERSION = "forst-invoke-v1";

export const RESERVED_INVOKE_HEADERS = [
  "x-forst-invoke-proof",
  "x-forst-invoke-generation",
  "x-forst-invoke-nonce",
  "x-forst-invoke-token",
] as const;

export function normalizeHeaders(
  headers: HeadersInit | undefined
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

export function stripReservedHeaders(
  headers: Record<string, string>
): Record<string, string> {
  const out: Record<string, string> = {};
  for (const [key, value] of Object.entries(headers)) {
    const lower = key.toLowerCase();
    if (RESERVED_INVOKE_HEADERS.includes(lower as (typeof RESERVED_INVOKE_HEADERS)[number])) {
      continue;
    }
    out[key] = value;
  }
  return out;
}

export function invokeProofMessage(generation: number, nonce: string): string {
  return `${INVOKE_PROOF_VERSION}|${generation}|${nonce}`;
}

export function computeInvokeProof(
  token: Uint8Array,
  generation: number,
  nonce: string
): string {
  return createHmac("sha256", token)
    .update(invokeProofMessage(generation, nonce))
    .digest("base64url");
}

export interface InvokeChallenge {
  nonce: string;
  generation: number;
  expiresAt: string;
}

export interface InvokeAuthState {
  token: Uint8Array;
  generation: number;
}

export interface InvokeReadyAuthPayload {
  url?: string;
  socketPath?: string;
  generation?: number;
  contractVersion?: string;
  runtime?: string;
}
