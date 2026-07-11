import {
  METHOD_CALL,
  METHOD_CALL_ASYNC,
  METHOD_GEN_CLOSE,
  METHOD_GEN_NEXT,
  METHOD_GEN_NEXT_BATCH,
  METHOD_GEN_OPEN,
  METHOD_GEN_RETURN,
  METHOD_INITIALIZE,
  METHOD_PING,
  METHOD_SHUTDOWN,
} from "../rpc/protocol.js";

/** Closed RPC method set from the spec — no dynamic registration. */
export const CLOSED_METHODS = new Set<string>([
  METHOD_INITIALIZE,
  METHOD_PING,
  METHOD_CALL,
  METHOD_CALL_ASYNC,
  METHOD_GEN_OPEN,
  METHOD_GEN_NEXT,
  METHOD_GEN_NEXT_BATCH,
  METHOD_GEN_RETURN,
  METHOD_GEN_CLOSE,
  METHOD_SHUTDOWN,
]);

/** Stub methods registered but rejected until later phases. */
export const STUB_METHODS = new Set<string>([
  METHOD_GEN_RETURN,
]);

export function isClosedMethod(method: string): boolean {
  return CLOSED_METHODS.has(method);
}

export function isStubMethod(method: string): boolean {
  return STUB_METHODS.has(method);
}
