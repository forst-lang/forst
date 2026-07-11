/** Supported RPC protocol major version (handshake). */
export const PROTOCOL_VERSION = 1;

export const WIRE_PROTOCOL_PROTO_V1 = "forst-node-proto-v1";

export const METHOD_INITIALIZE = "forst.node/initialize";
export const METHOD_PING = "forst.node/ping";
export const METHOD_CALL = "forst.node/call";
export const METHOD_CALL_ASYNC = "forst.node/callAsync";
export const METHOD_GEN_OPEN = "forst.node/genOpen";
export const METHOD_GEN_NEXT = "forst.node/genNext";
export const METHOD_GEN_NEXT_BATCH = "forst.node/genNextBatch";
export const METHOD_GEN_RETURN = "forst.node/genReturn";
export const METHOD_GEN_CLOSE = "forst.node/genClose";
export const METHOD_SHUTDOWN = "forst.node/shutdown";

export type JsonRpcId = string | number | null;

export interface JsonRpcRequest {
  jsonrpc: "2.0";
  id?: JsonRpcId;
  method: string;
  params?: unknown;
}

export interface JsonRpcSuccessResponse {
  jsonrpc: "2.0";
  id: JsonRpcId;
  result: unknown;
}

export interface JsonRpcErrorResponse {
  jsonrpc: "2.0";
  id: JsonRpcId;
  error: {
    code: number;
    message: string;
    data?: unknown;
  };
}

export type JsonRpcResponse = JsonRpcSuccessResponse | JsonRpcErrorResponse;

export interface InitializeParams {
  protocolVersion: number;
  boundaryRoot: string;
  manifest: unknown;
  filesExclude?: string[];
  supportedProtocols?: string[];
}

export interface InitializeResult {
  ok: true;
  protocol: string;
}

export interface PingResult {
  pong: true;
}

export interface CallParams {
  moduleId: string;
  exportName: string;
  args?: unknown[];
}

export interface CallResult {
  value: unknown;
}

export interface GenOpenParams {
  moduleId: string;
  exportName: string;
  args?: unknown[];
}

export interface GenOpenResult {
  streamId: string;
}

export interface GenNextParams {
  streamId: string;
}

export interface GenNextBatchParams {
  streamId: string;
  maxItems?: number;
}

export interface GenNextBatchResult {
  steps: GenNextResult[];
}

export type GenNextKind = "yield" | "done" | "error";

export interface GenNextResult {
  kind: GenNextKind;
  value?: unknown;
  message?: string;
  data?: unknown;
}

export interface GenCloseParams {
  streamId: string;
}

export interface GenCloseResult {
  ok: true;
}

export interface ShutdownResult {
  ok: true;
}
