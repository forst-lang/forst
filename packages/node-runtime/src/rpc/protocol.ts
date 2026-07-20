/** Supported RPC protocol major version (handshake). */
export const PROTOCOL_VERSION = 1;

/** Wire protocol identifier negotiated during initialize (length-prefixed protobuf v1). */
export const WIRE_PROTOCOL_PROTO_V1 = "forst-node-proto-v1";

/** RPC method: initialize runtime with manifest and boundary root. */
export const METHOD_INITIALIZE = "forst.node/initialize";
/** RPC method: health check ping. */
export const METHOD_PING = "forst.node/ping";
/** RPC method: synchronous export call. */
export const METHOD_CALL = "forst.node/call";
/** RPC method: fire-and-forget export call (returns before async work completes). */
export const METHOD_CALL_ASYNC = "forst.node/callAsync";
/** RPC method: open an async generator stream over a module export. */
export const METHOD_GEN_OPEN = "forst.node/genOpen";
/** RPC method: pull the next yield/done/error step from an open generator stream. */
export const METHOD_GEN_NEXT = "forst.node/genNext";
/** RPC method: pull a batch of generator steps in one round trip. */
export const METHOD_GEN_NEXT_BATCH = "forst.node/genNextBatch";
/** RPC method: return a value from a generator stream (Effect-style completion). */
export const METHOD_GEN_RETURN = "forst.node/genReturn";
/** RPC method: close and release a generator stream by id. */
export const METHOD_GEN_CLOSE = "forst.node/genClose";
/** RPC method: shut down the runtime. */
export const METHOD_SHUTDOWN = "forst.node/shutdown";

/** JSON-RPC request identifier. */
export type JsonRpcId = string | number | null;

/** Incoming JSON-RPC 2.0 request. */
export interface JsonRpcRequest {
  /** Protocol version marker. */
  jsonrpc: "2.0";
  /** Request id; omitted for notifications. */
  id?: JsonRpcId;
  /** Method name. */
  method: string;
  /** Method parameters. */
  params?: unknown;
}

/** JSON-RPC success envelope returned to the Go host after a handled method. */
export interface JsonRpcSuccessResponse {
  jsonrpc: "2.0";
  id: JsonRpcId;
  result: unknown;
}

/** JSON-RPC error envelope returned when dispatch or parsing fails. */
export interface JsonRpcErrorResponse {
  jsonrpc: "2.0";
  id: JsonRpcId;
  error: {
    code: number;
    message: string;
    data?: unknown;
  };
}

/** JSON-RPC success or error response. */
export type JsonRpcResponse = JsonRpcSuccessResponse | JsonRpcErrorResponse;

/** Parameters for the initialize handshake (manifest, boundary root, protocol version). */
export interface InitializeParams {
  protocolVersion: number;
  boundaryRoot: string;
  manifest: unknown;
  filesExclude?: string[];
  supportedProtocols?: string[];
}

/** Result of a successful initialize: confirms the negotiated wire protocol string. */
export interface InitializeResult {
  ok: true;
  protocol: string;
}

/** Result of a ping health check. */
export interface PingResult {
  pong: true;
}

/** Parameters for sync/async module export calls. */
export interface CallParams {
  moduleId: string;
  exportName: string;
  args?: unknown[];
}

/** Result of a synchronous export call. */
export interface CallResult {
  value: unknown;
}

/** Parameters for opening a generator stream over a module export. */
export interface GenOpenParams {
  moduleId: string;
  exportName: string;
  args?: unknown[];
}

/** Result of genOpen: opaque stream id used by subsequent genNext/genClose calls. */
export interface GenOpenResult {
  streamId: string;
}

/** Parameters for pulling one step from an open generator stream. */
export interface GenNextParams {
  streamId: string;
}

/** Parameters for batched generator pulls (reduces round trips on the bridge). */
export interface GenNextBatchParams {
  streamId: string;
  maxItems?: number;
}

/** Batch result wrapping multiple generator steps from one RPC. */
export interface GenNextBatchResult {
  steps: GenNextResult[];
}

/** Discriminator for a single generator step on the wire. */
export type GenNextKind = "yield" | "done" | "error";

/** One generator step: yielded value, completion, or stream error. */
export interface GenNextResult {
  kind: GenNextKind;
  value?: unknown;
  message?: string;
  data?: unknown;
}

/** Parameters for closing a generator stream and freeing server-side state. */
export interface GenCloseParams {
  streamId: string;
}

/** Acknowledgement that a generator stream was closed. */
export interface GenCloseResult {
  ok: true;
}

/** Acknowledgement that the runtime accepted shutdown. */
export interface ShutdownResult {
  ok: true;
}
