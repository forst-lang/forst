/**
 * Re-exports invoke HMAC helpers from `@forst/cli/invoke` (shared source of truth).
 */
export {
  INVOKE_PROOF_VERSION,
  RESERVED_INVOKE_HEADERS,
  authDisabledByEnv,
  computeInvokeProof,
  invokeProofMessage,
  normalizeHeaders,
  parseInvokeChallengeResult,
  stripReservedHeaders,
  warnIfInvokeAuthDisabled,
  type InvokeAuthState,
  type InvokeChallenge,
  type InvokeHeadersInit,
} from "@forst/cli/invoke";

/** Ready-file metadata shape (no token). Alias kept for sidecar call sites. */
export type { InvokeReadyPayload as InvokeReadyAuthPayload } from "@forst/cli/invoke";
