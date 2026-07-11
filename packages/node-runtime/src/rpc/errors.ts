/** JSON-RPC 2.0 standard error codes. */
export const PARSE_ERROR = -32700;
export const INVALID_REQUEST = -32600;
export const METHOD_NOT_FOUND = -32601;
export const INVALID_PARAMS = -32602;
export const INTERNAL_ERROR = -32603;

/** Forst node-runtime application error codes. */
export const APPLICATION_ERROR = -32000;
export const FORBIDDEN = -32001;
export const NOT_INITIALIZED = -32002;
export const NOT_IMPLEMENTED = -32003;

export interface JsonRpcErrorObject {
  code: number;
  message: string;
  data?: unknown;
}

export class JsonRpcError extends Error {
  readonly code: number;
  readonly data: unknown | undefined;

  constructor(code: number, message: string, data?: unknown) {
    super(message);
    this.name = "JsonRpcError";
    this.code = code;
    this.data = data;
    Object.setPrototypeOf(this, new.target.prototype);
  }

  toObject(): JsonRpcErrorObject {
    const obj: JsonRpcErrorObject = { code: this.code, message: this.message };
    if (this.data !== undefined) {
      obj.data = this.data;
    }
    return obj;
  }
}

export function forbidden(message: string, data?: unknown): JsonRpcError {
  return new JsonRpcError(FORBIDDEN, message, data);
}

export function notInitialized(message = "Runtime not initialized"): JsonRpcError {
  return new JsonRpcError(NOT_INITIALIZED, message);
}

export function notImplemented(method: string): JsonRpcError {
  return new JsonRpcError(NOT_IMPLEMENTED, `Method not implemented: ${method}`);
}

export function invalidParams(message: string, data?: unknown): JsonRpcError {
  return new JsonRpcError(INVALID_PARAMS, message, data);
}

export function internalError(message: string, data?: unknown): JsonRpcError {
  return new JsonRpcError(INTERNAL_ERROR, message, data);
}

export function applicationError(
  message: string,
  data?: Record<string, unknown>
): JsonRpcError {
  return new JsonRpcError(APPLICATION_ERROR, message, data);
}
