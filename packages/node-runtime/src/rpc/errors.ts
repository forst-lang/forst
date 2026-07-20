import { Data } from "effect";

/** JSON-RPC parse failure (-32700); returned when the request body is not valid JSON. */
export const PARSE_ERROR = -32700;
/** JSON-RPC invalid request (-32600); returned when the envelope shape is wrong. */
export const INVALID_REQUEST = -32600;
/** JSON-RPC method not found (-32601); returned when no handler recognizes the method name. */
export const METHOD_NOT_FOUND: number = -32601;
/** JSON-RPC invalid params (-32602); returned when params fail schema or policy checks. */
export const INVALID_PARAMS = -32602;
/** JSON-RPC internal error (-32603); returned for unexpected runtime failures on the wire. */
export const INTERNAL_ERROR = -32603;

/** Forst application-level failure (-32000); returned for domain errors surfaced to the Go host. */
export const APPLICATION_ERROR = -32000;
/** Export or module access denied by manifest policy (-32001). */
export const FORBIDDEN: number = -32001;
/** RPC issued before initialize completed (-32002); guards calls against an uninitialized runtime. */
export const NOT_INITIALIZED: number = -32002;
/** RPC method recognized but not implemented in this build (-32003). */
export const NOT_IMPLEMENTED: number = -32003;

/** Plain JSON-RPC error object shape written into proto frame error responses. */
export interface JsonRpcErrorObject {
  code: number;
  message: string;
  data?: unknown;
}

/** Tagged Effect error carrying a JSON-RPC code; converted to wire responses, not thrown across the loop. */
export class JsonRpcError extends Data.TaggedError("JsonRpcError")<{
  readonly code: number;
  readonly message: string;
  readonly data?: unknown;
}> {
  toObject(): JsonRpcErrorObject {
    const obj: JsonRpcErrorObject = { code: this.code, message: this.message };
    if (this.data !== undefined) {
      obj.data = this.data;
    }
    return obj;
  }
}

/** Returns a policy-denied error response (manifest blocked the export or module). */
export function forbidden(message: string, data?: unknown): JsonRpcError {
  return new JsonRpcError({ code: FORBIDDEN, message, data });
}

/** Returns an error when a non-initialize RPC arrives before the runtime handshake completes. */
export function notInitialized(message = "Runtime not initialized"): JsonRpcError {
  return new JsonRpcError({ code: NOT_INITIALIZED, message });
}

/** Returns an error for methods declared in policy but not yet implemented in this build. */
export function notImplemented(method: string): JsonRpcError {
  return new JsonRpcError({
    code: NOT_IMPLEMENTED,
    message: `Method not implemented: ${method}`,
  });
}

/** Returns an error when RPC params fail validation or cannot be coerced. */
export function invalidParams(message: string, data?: unknown): JsonRpcError {
  return new JsonRpcError({ code: INVALID_PARAMS, message, data });
}

/** Returns an error for unexpected handler failures that should map to JSON-RPC internal error. */
export function internalError(message: string, data?: unknown): JsonRpcError {
  return new JsonRpcError({ code: INTERNAL_ERROR, message, data });
}

/** Returns a domain-level application error with optional structured data for the Go host. */
export function applicationError(
  message: string,
  data?: Record<string, unknown>
): JsonRpcError {
  return new JsonRpcError({ code: APPLICATION_ERROR, message, data });
}

/** Returns an error when the JSON-RPC request envelope is malformed. */
export function invalidRequest(message: string): JsonRpcError {
  return new JsonRpcError({ code: INVALID_REQUEST, message });
}

/** Returns an error when a decoded proto frame has no request half (client sent a response-only frame). */
export function frameMissingRequest(): JsonRpcError {
  return invalidRequest("frame missing request");
}

/** Returns an error when the wire request omits or empties the JSON-RPC method field. */
export function requestMethodRequired(): JsonRpcError {
  return invalidRequest("method must be a string");
}

/** Returns a parse error when request payload bytes are not valid JSON. */
export function invalidRequestPayloadJson(): JsonRpcError {
  return new JsonRpcError({
    code: PARSE_ERROR,
    message: "invalid JSON in request payload",
  });
}

/** Returns a generic JSON-RPC parse error for framing or envelope failures. */
export function parseError(message = "parse error"): JsonRpcError {
  return new JsonRpcError({ code: PARSE_ERROR, message });
}

/** Returns an error when the method name is not registered in the closed RPC surface. */
export function methodNotFound(message = "Method not found"): JsonRpcError {
  return new JsonRpcError({ code: METHOD_NOT_FOUND, message });
}

/** Rehydrates a {@link JsonRpcError} from a plain object (e.g. when bridging error responses). */
export function fromJsonRpcErrorObject(error: JsonRpcErrorObject): JsonRpcError {
  return new JsonRpcError({
    code: error.code,
    message: error.message,
    data: error.data,
  });
}
