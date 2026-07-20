import { Data } from "effect";

/** JSON-RPC 2.0 standard error codes. */
export const PARSE_ERROR = -32700;
export const INVALID_REQUEST = -32600;
/** JSON-RPC method not found. */
export const METHOD_NOT_FOUND: number = -32601;
export const INVALID_PARAMS = -32602;
export const INTERNAL_ERROR = -32603;

/** Forst node-runtime application error codes. */
export const APPLICATION_ERROR = -32000;
/** Export or module access denied by manifest policy. */
export const FORBIDDEN: number = -32001;
/** RPC issued before initialize completed. */
export const NOT_INITIALIZED: number = -32002;
/** RPC method recognized but not implemented in this build. */
export const NOT_IMPLEMENTED: number = -32003;

export interface JsonRpcErrorObject {
  code: number;
  message: string;
  data?: unknown;
}

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

export function forbidden(message: string, data?: unknown): JsonRpcError {
  return new JsonRpcError({ code: FORBIDDEN, message, data });
}

export function notInitialized(message = "Runtime not initialized"): JsonRpcError {
  return new JsonRpcError({ code: NOT_INITIALIZED, message });
}

export function notImplemented(method: string): JsonRpcError {
  return new JsonRpcError({
    code: NOT_IMPLEMENTED,
    message: `Method not implemented: ${method}`,
  });
}

export function invalidParams(message: string, data?: unknown): JsonRpcError {
  return new JsonRpcError({ code: INVALID_PARAMS, message, data });
}

export function internalError(message: string, data?: unknown): JsonRpcError {
  return new JsonRpcError({ code: INTERNAL_ERROR, message, data });
}

export function applicationError(
  message: string,
  data?: Record<string, unknown>
): JsonRpcError {
  return new JsonRpcError({ code: APPLICATION_ERROR, message, data });
}

export function invalidRequest(message: string): JsonRpcError {
  return new JsonRpcError({ code: INVALID_REQUEST, message });
}

export function frameMissingRequest(): JsonRpcError {
  return invalidRequest("frame missing request");
}

export function requestMethodRequired(): JsonRpcError {
  return invalidRequest("method must be a string");
}

export function invalidRequestPayloadJson(): JsonRpcError {
  return new JsonRpcError({
    code: PARSE_ERROR,
    message: "invalid JSON in request payload",
  });
}

export function parseError(message = "parse error"): JsonRpcError {
  return new JsonRpcError({ code: PARSE_ERROR, message });
}

export function methodNotFound(message = "Method not found"): JsonRpcError {
  return new JsonRpcError({ code: METHOD_NOT_FOUND, message });
}

export function fromJsonRpcErrorObject(error: JsonRpcErrorObject): JsonRpcError {
  return new JsonRpcError({
    code: error.code,
    message: error.message,
    data: error.data,
  });
}
